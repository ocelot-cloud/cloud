package cloud

import (
	"archive/zip"
	"errors"
	"fmt"
	"github.com/ocelot-cloud/shared/utils"
	"github.com/ocelot-cloud/shared/validation"
	"gopkg.in/yaml.v3"
	"io"
	"ocelot/backend/apps/common"
	"ocelot/backend/clients"
	"ocelot/backend/tools"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

func GetAppManager() clients.AppManager {
	if tools.AreMocksUsed() {
		return &clients.MockAppManager{}
	} else {
		return &RealAppManager{}
	}
}

type RealAppManager struct{}

var cantHaveTwoAppsWithSameNameRunningAtSameTime = "can't have two apps with the same name running at the same time"

func (r *RealAppManager) StartApp(appId int) error {
	defer UpdateAppConfigs()
	app, err := common.AppRepo.GetApp(appId)
	if err != nil {
		return err
	}

	isThereAnotherAppWithSameNameRunning, err := isAnotherAppWithSameNameRunningAtSameTime(app)
	if err != nil {
		return err
	} else if isThereAnotherAppWithSameNameRunning {
		return errors.New(cantHaveTwoAppsWithSameNameRunningAtSameTime)
	}

	CreateExternalDockerNetworkAndConnectOcelotCloud(*app)

	dockerStackName := app.Maintainer + "_" + app.AppName
	cmd := exec.Command("docker", "compose", "-p", dockerStackName, "up", "-d") // #nosec G204 (CWE-78): Execution as root with variables in subprocess is required by design
	err = extractVersionZipToDirAndDeploy(app.VersionContent, cmd, app.Maintainer, app.AppName)
	if err != nil {
		return err
	}
	err = common.AppRepo.SetAppShouldBeRunning(appId, true)
	if err != nil {
		return err
	}
	return nil
}

func isAnotherAppWithSameNameRunningAtSameTime(app *tools.RepoApp) (bool, error) {
	apps, err := common.AppRepo.ListApps()
	if err != nil {
		return true, err
	}
	for _, otherApp := range apps {
		if otherApp.AppName == app.AppName && otherApp.Maintainer != app.Maintainer && otherApp.ShouldBeRunning {
			return true, nil
		}
	}

	return false, nil
}

func CreateExternalDockerNetworkAndConnectOcelotCloud(app tools.RepoApp) {
	networkName := app.Maintainer + "_" + app.AppName
	networkCreationCmd := fmt.Sprintf("docker network ls | grep -q %s || docker network create %s", networkName, networkName)
	err := utils.ExecuteShellCommand(networkCreationCmd)
	if err != nil {
		Logger.Fatal("Failed to create network: %v", err)
	}
	ConnectOcelotCloudWithNetwork(networkName)
}

func ConnectOcelotCloudWithNetwork(networkName string) {
	if !tools.Config.IsUsingDockerNetwork {
		return
	}
	networkConnectionCmd := fmt.Sprintf("docker network connect %s ocelotcloud", networkName)
	err := utils.ExecuteShellCommand(networkConnectionCmd)
	if err == nil {
		Logger.Info("Connected ocelotcloud to network '%s'", networkName)
	} else {
		Logger.Debug("Failed to connect ocelotcloud to network '%s', usually not critical as it appears because ocelotcloud is already connected: %v", networkName, err)
	}
}

func unzip(srcFilePath string, destinationDir string) error {
	reader, err := zip.OpenReader(srcFilePath)
	if err != nil {
		Logger.Error("Failed to open zip file: %v", err)
		return err
	}
	defer utils.Close(reader)

	for _, file := range reader.File {
		filePath := filepath.Join(destinationDir, file.Name) // #nosec G305 (CWE-22): File traversal when extracting zip/tar archive; is okay, since path is generated internally
		if !strings.HasPrefix(filePath, filepath.Clean(destinationDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", filePath)
		}

		if file.FileInfo().IsDir() {
			err = os.MkdirAll(filePath, file.Mode())
			if err != nil {
				Logger.Error("Failed to create directory: %v", err)
				return err
			}
			continue
		}

		err = os.MkdirAll(filepath.Dir(filePath), file.Mode())
		if err != nil {
			Logger.Error("Failed to create directory: %v", err)
			return err
		}

		outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode()) // #nosec G304 (CWE-22): Potential file inclusion via variable; is okay, since path is generated internally
		if err != nil {
			Logger.Error("Failed to open file: %v", err)
			return err
		}
		defer utils.Close(outFile)

		rc, err := file.Open()
		if err != nil {
			Logger.Error("Failed to open file in zip: %v", err)
			return err
		}
		defer utils.Close(rc)

		_, err = io.Copy(outFile, rc) // #nosec G110 (CWE-409): Potential DoS vulnerability via decompression bomb; is already checked by app store
		if err != nil {
			Logger.Error("Failed to copy file: %v", err)
			return err
		}
	}
	return nil
}

type AppAndTagIds struct {
	AppId  string `json:"app_id"`
	TagIds string `json:"version_id"`
}

func (r *RealAppManager) StopApp(appId int) error {
	err := common.AppRepo.SetAppShouldBeRunning(appId, false)
	if err != nil {
		return err
	}
	app, err := common.AppRepo.GetApp(appId)
	if err != nil {
		return err
	}

	dockerStackName := app.Maintainer + "_" + app.AppName
	cmd := exec.Command("docker", "compose", "-p", dockerStackName, "down") // #nosec G204 (CWE-78): Execution as root with variables in subprocess is required by design
	err = extractVersionZipToDirAndDeploy(app.VersionContent, cmd, app.Maintainer, app.AppName)
	if err != nil {
		return err
	}

	return nil
}

func extractVersionZipToDir(content []byte) (string, error) {
	tempDir, err := os.MkdirTemp("", "docker-compose")
	if err != nil {
		Logger.Error("Failed to create temp directory: %v", err)
		return "", err
	}

	zipFilePath := filepath.Join(tempDir, "archive.zip")
	err = os.WriteFile(zipFilePath, content, 0600)
	if err != nil {
		err2 := os.RemoveAll(tempDir)
		if err2 != nil {
			Logger.Error("Failed to delete temp directory: %v", err2)
		}
		Logger.Error("Failed to write zip file: %v", err)
		return "", err
	}

	err = unzip(zipFilePath, tempDir)
	if err != nil {
		err2 := os.RemoveAll(tempDir)
		if err2 != nil {
			Logger.Error("Failed to delete temp directory: %v", err2)
		}
		Logger.Error("Failed to unzip file: %v", err)
		return "", err
	}

	return tempDir, nil
}

func extractVersionZipToDirAndDeploy(content []byte, command *exec.Cmd, maintainer, appName string) error {
	tempDir, err := extractVersionZipToDir(content)
	if err != nil {
		return err
	}
	defer utils.RemoveDir(tempDir)

	if !common.IsOcelotDb(maintainer, appName) {
		err = validation.CompleteDockerComposeYaml(maintainer, appName, tempDir+"/docker-compose.yml")
		if err != nil {
			return err
		}
	}

	cmd := command
	cmd.Dir = tempDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		Logger.Error("Failed to run command: %v", err)
		return err
	}
	return nil
}

type AppConfig struct {
	Port    int    `yaml:"port"`
	UrlPath string `yaml:"url_path"`
}

var (
	appConfigsMu sync.RWMutex
	appConfigs   map[string]AppConfig
)

func GetAppConfig(appName string) (AppConfig, bool) {
	appConfigsMu.RLock()
	defer appConfigsMu.RUnlock()
	cfg, ok := appConfigs[appName]
	return cfg, ok
}

func UpdateAppConfigs() {
	Logger.Info("reading app configs from the database")
	appConfigsMu.Lock()
	defer appConfigsMu.Unlock()
	newAppConfigs := make(map[string]AppConfig)

	apps, err := common.AppRepo.ListApps()
	if err != nil {
		Logger.Error("Failed to list apps: %v", err)
		return
	}

	for _, app := range apps {
		if !app.ShouldBeRunning {
			continue
		}
		if common.IsOcelotDbApp(app) {
			continue
		}
		path, err := extractVersionZipToDir(app.VersionContent)
		defer deletePath(path)
		if err != nil {
			Logger.Error("Failed to extract tag content: %v", err)
		}
		Logger.Debug("Extracting tag to path: %s", path)

		appConfig, err := readAppConfig(path)
		if err != nil {
			Logger.Error("Failed to read app config: %v", err)
		}
		newAppConfigs[app.AppName] = *appConfig
	}
	appConfigs = newAppConfigs
}

func deletePath(path string) {
	if err := os.RemoveAll(path); err != nil {
		Logger.Error("Failed to delete path: %v", err)
	}
}

func readAppConfig(path string) (*AppConfig, error) {
	var config AppConfig
	data, err := os.ReadFile(path + "/app.yml") // #nosec G304 (CWE-22): Potential file inclusion via variable; is okay, since path is generated internally
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		Logger.Error("Error reading file: %v", err)
		return nil, fmt.Errorf("error reading file")
	}

	if errors.Is(err, os.ErrNotExist) {
		return &AppConfig{Port: 80, UrlPath: "/"}, nil
	} else {
		if err := yaml.Unmarshal(data, &config); err != nil {
			Logger.Error("Error unmarshalling YAML: %v", err)
			return nil, fmt.Errorf("error unmarshalling YAML")
		}
	}

	if config.Port == 0 {
		config.Port = 80
	}
	if config.UrlPath == "" {
		config.UrlPath = "/"
	}

	return &config, nil
}

// "isIndexPageAvailable" is the parameter that will be determined by the healthcheck feature to be implemented
func getStatus(shouldBeRunning, isIndexPageAvailable bool) string {
	if shouldBeRunning {
		if isIndexPageAvailable {
			return "Available"
		} else {
			return "Starting"
		}
	} else {
		return "Uninitialized"
	}
}
