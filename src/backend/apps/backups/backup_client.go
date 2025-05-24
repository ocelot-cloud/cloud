package backups

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ocelot-cloud/shared/utils"
	"gopkg.in/yaml.v3"
	"io"
	"ocelot/backend/apps/cloud"
	"ocelot/backend/apps/common"
	"ocelot/backend/clients"
	"ocelot/backend/ssh"
	"ocelot/backend/tools"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	numberOfDailyBackupsToKeep   = 7
	numberOfWeeklyBackupsToKeep  = 4
	numberOfMonthlyBackupsToKeep = 12
)

// for developers: set to true to print command output to console
const showCommandOutput = false

type RealBackupManager struct{}

func (b *RealBackupManager) CreateBackup(appId int, description tools.BackupDescription) error {
	defer cloud.UpdateAppConfigs()
	err := b.createBackupAtLocation(appId, description, true)
	if err != nil {
		return err
	}
	isRemoteBackupEnabled, err := ssh.IsRemoteBackupEnabled()
	if err != nil {
		return err
	}
	if isRemoteBackupEnabled {
		return b.createBackupAtLocation(appId, description, false)
	} else {
		return nil
	}
}

var (
	backupRepositoryPathInResticContainer = "/backups"
	backupDockerVolumeName                = "backups"

	localBackupResticCommandEnvs = []string{
		`RESTIC_REPOSITORY=/backups`,
		// all restic backups must have a password and empty passwords are not allowed, so we chose this as default
		`RESTIC_PASSWORD=password`,
	}
)

func getBackupCreationDto(appId int, description tools.BackupDescription) (BackupCreationDto, error) {
	app, err := common.AppRepo.GetApp(appId)
	if err != nil {
		Logger.Error("Error getting app")
		return BackupCreationDto{}, err
	}

	backupCreation := BackupCreationDto{
		Maintainer:               app.Maintainer,
		AppName:                  app.AppName,
		VersionName:              app.VersionName,
		VersionCreationTimestamp: app.VersionCreationTimestamp.Format(time.RFC3339),
		Description:              string(description),
		VersionZipContent:        app.VersionContent,
	}
	return backupCreation, nil
}

func (b *RealBackupManager) createBackupAtLocation(appId int, description tools.BackupDescription, isLocalBackup bool) error {
	backupCreationDto, err := getBackupCreationDto(appId, description)
	if err != nil {
		return err
	}

	volumes, err := extractVolumesFromZipsDockerComposeYaml(backupCreationDto.VersionZipContent)
	if err != nil {
		return err
	}

	resticTags := []string{
		"maintainer=" + backupCreationDto.Maintainer,
		"app=" + backupCreationDto.AppName,
		"version=" + backupCreationDto.VersionName,
		"version_creation_timestamp=" + backupCreationDto.VersionCreationTimestamp,
		"description=" + backupCreationDto.Description,
	}

	app, err := common.AppRepo.GetApp(appId)
	if err != nil {
		return err
	}
	envs, err := prepareResticOperationAndReturnCommandEnvs(isLocalBackup)
	if err != nil {
		return err
	}

	err = clients.Apps.StopApp(appId)
	if err != nil {
		return err
	}

	tempDir, zipName, err := createZipFile(backupCreationDto)
	if err != nil {
		return err
	}
	defer utils.RemoveDir(tempDir)
	zipFileMountVolume := fmt.Sprintf("-v %s/%s:/source/%s ", tempDir, zipName, zipName)

	_, err = executeInResticContainer("restic backup /source", volumes, resticTags, envs, zipFileMountVolume)
	if err != nil {
		return err
	}

	if common.IsOcelotDbApp(*app) {
		common.InitializeDatabase(tools.Config.IsUsingDockerNetwork, tools.Config.UseProductionDatabaseContainer)
		err = common.AppRepo.SetAppShouldBeRunning(appId, true)
		if err != nil {
			Logger.Error("Error setting postgres app should be running")
			return err
		}
	} else {
		err = clients.Apps.StartApp(appId)
		if err != nil {
			return err
		}
	}

	return nil
}

func prepareResticOperationAndReturnCommandEnvs(isLocalBackup bool) ([]string, error) {
	if isLocalBackup {
		_, err := executeInResticContainer("restic check || restic init", nil, nil, localBackupResticCommandEnvs, "")
		if err != nil {
			return nil, err
		}
		return localBackupResticCommandEnvs, nil
	} else {
		repository, err := ssh.GetRemoteBackupRepository()
		if err != nil {
			return nil, err
		}
		knownHostsFileLocation := "/root/.ssh/known_hosts"
		rcloneSetupCmd := fmt.Sprintf("rclone config create myssh sftp host=%s user=%s pass=%s port=%s known_hosts_file=%s use_insecure_cipher=false", repository.Host, repository.SshUser, repository.SshPassword, repository.SshPort, knownHostsFileLocation)
		_, err = executeInResticContainer(rcloneSetupCmd, nil, nil, nil, "")
		if err != nil {
			return nil, err
		}

		knownHostsSetupCmd := fmt.Sprintf("printf '%s' > %s", repository.SshKnownHosts, knownHostsFileLocation)
		_, err = executeInResticContainer(knownHostsSetupCmd, nil, nil, nil, "")
		if err != nil {
			return nil, err
		}

		envs := []string{
			"RESTIC_REPOSITORY=rclone:myssh:backups",
			"RESTIC_PASSWORD=" + repository.EncryptionPassword,
		}
		_, err = executeInResticContainer("restic check || restic init", nil, nil, envs, "")
		if err != nil {
			return nil, err
		}

		return envs, nil
	}
}

func createZipFile(backup BackupCreationDto) (string, string, error) {
	tempDir, err := os.MkdirTemp(tools.TempDir, "temp")
	if err != nil {
		return "", "", err
	}
	fileName := backup.VersionName + ".zip"
	filePath := filepath.Join(tempDir, backup.VersionName+".zip")
	err = os.WriteFile(filePath, backup.VersionZipContent, 0600)
	if err != nil {
		return "", "", err
	}
	return tempDir, fileName, nil
}

func executeInResticContainer(command string, appVolumes, resticTags, envs []string, mountVolume string) (string, error) {
	resticTagsFlags := ""
	for _, tag := range resticTags {
		resticTagsFlags += `--tag ` + tag + ` `
	}
	volumeFlags := ""
	for _, volume := range appVolumes {
		volumeFlags += `-v ` + volume + `:/source/` + volume + ` `
	}
	envFlags := ""
	for _, env := range envs {
		envFlags += `-e ` + env + ` `
	}

	wholeCommand := fmt.Sprintf(`docker run --rm --network %s %s-v %s:%s %s%s--entrypoint "" -v restic_rclone:/root/.config/rclone -v restic_ssh:/root/.ssh restic:local sh -c "%s %s"`, tools.ResticDockerNetwork, mountVolume, backupDockerVolumeName, backupRepositoryPathInResticContainer, volumeFlags, envFlags, command, resticTagsFlags)
	return runCommandWithOutputString(wholeCommand)
}

func extractVolumesFromZipsDockerComposeYaml(zipContent []byte) ([]string, error) {
	tempDir, err := utils.UnzipToTempDir(zipContent)
	if err != nil {
		return nil, err
	}
	defer utils.RemoveDir(tempDir)

	data, err := os.ReadFile(tempDir + "/docker-compose.yml") // #nosec G304 (CWE-22): Potential file inclusion via variable; is okay, since path is generated internally
	if err != nil {
		return nil, err
	}

	var config DockerComposeYaml
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	var volumes []string
	for key := range config.Volumes {
		volumes = append(volumes, key)
	}
	return volumes, nil
}

func (b *RealBackupManager) ListBackupsOfApp(backupListRequest tools.BackupListRequest) ([]tools.BackupInfo, error) {
	resticTagFilters := []string{
		"maintainer=" + backupListRequest.Maintainer,
		"app=" + backupListRequest.AppName,
	}

	backupInfos, err := b.getBackupsMatchingTags(backupListRequest.IsLocal, resticTagFilters)
	if err != nil {
		return nil, err
	}

	var filteredBackupInfos []tools.BackupInfo
	for _, backupInfo := range backupInfos {
		if backupInfo.Maintainer == backupListRequest.Maintainer && backupInfo.AppName == backupListRequest.AppName {
			filteredBackupInfos = append(filteredBackupInfos, backupInfo)
		}
	}

	return filteredBackupInfos, nil
}

func (b *RealBackupManager) getBackupsMatchingTags(useLocalBackupRepo bool, resticTagFilters []string) ([]tools.BackupInfo, error) {
	isRemoteRepoEnabled, err := ssh.IsRemoteBackupEnabled()
	if err != nil {
		return nil, err
	}
	if !useLocalBackupRepo && !isRemoteRepoEnabled {
		Logger.Info("remote backup repository is not enabled, skipping backup listing")
		return nil, nil
	}

	envs, err := prepareResticOperationAndReturnCommandEnvs(useLocalBackupRepo)
	if err != nil {
		return nil, err
	}
	output, err := executeInResticContainer("restic snapshots --json", nil, resticTagFilters, envs, "")
	if err != nil {
		return nil, err
	}
	backupInfos, err := parseBackupInfo(output)
	if err != nil {
		return nil, err
	}
	return backupInfos, nil
}

func parseBackupInfo(jsonStr string) ([]tools.BackupInfo, error) {
	var snapshots []Snapshot
	if err := json.Unmarshal([]byte(jsonStr), &snapshots); err != nil {
		return nil, err
	}

	var backups []tools.BackupInfo
	for _, snap := range snapshots {
		parsedTime, err := time.Parse(time.RFC3339, snap.Time)
		if err != nil {
			return nil, fmt.Errorf("failed to parse time: %v", err)
		}

		tagMap := make(map[string]string)
		for _, tag := range snap.Tags {
			parts := strings.SplitN(tag, "=", 2)
			if len(parts) == 2 {
				tagMap[parts[0]] = parts[1]
			}
		}

		s := tagMap["version_creation_timestamp"]
		versionCreationTimestamp, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return nil, fmt.Errorf("failed to parse time: %v", err)
		}

		backups = append(backups, tools.BackupInfo{
			BackupId:                 snap.Id,
			Maintainer:               tagMap["maintainer"],
			AppName:                  tagMap["app"],
			VersionName:              tagMap["version"],
			VersionCreationTimestamp: versionCreationTimestamp.UTC(),
			Description:              tools.BackupDescription(tagMap["description"]),
			BackupCreationTimestamp:  parsedTime.UTC(),
		})
	}

	return backups, nil
}

func (b *RealBackupManager) DeleteBackup(backupId string, isLocalBackup bool) error {
	envs, err := prepareResticOperationAndReturnCommandEnvs(isLocalBackup)
	if err != nil {
		return err
	}
	_, err = executeInResticContainer("restic forget "+backupId, nil, nil, envs, "")
	if err != nil {
		return err
	}
	return nil
}

func (b *RealBackupManager) RestoreBackup(request tools.BackupOperationRequest) (*tools.RestoredVersionInfo, error) {
	defer cloud.UpdateAppConfigs()
	envs, err := prepareResticOperationAndReturnCommandEnvs(request.IsLocal)
	if err != nil {
		return nil, err
	}

	zipFileContent, volumes, err := b.fetchAndPrepareZip(request.BackupId, envs)
	if err != nil {
		return nil, err
	}

	err = b.cleanupAndRestoreVolumes(request.BackupId, volumes, envs)
	if err != nil {
		return nil, err
	}

	rvi, err := b.finalizeRestore(request.BackupId, zipFileContent, envs)
	if err != nil {
		return nil, err
	}

	return rvi, nil
}

func (b *RealBackupManager) fetchAndPrepareZip(backupId string, envs []string) ([]byte, []string, error) {
	tempDir, err := os.MkdirTemp(tools.TempDir, "temp")
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err != nil {
			err2 := os.RemoveAll(tempDir)
			if err2 != nil {
				Logger.Error("Failed to delete temp directory: %v", err2)
			}
		}
	}()

	_, err = executeInResticContainer(
		"restic restore "+backupId+" --target / --include '*.zip'",
		nil, nil, envs,
		"-v "+tempDir+":/source ",
	)
	if err != nil {
		return nil, nil, err
	}

	files, err := os.ReadDir(tempDir)
	if err != nil {
		return nil, nil, err
	}
	var zipPath string
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".zip" {
			zipPath = filepath.Join(tempDir, f.Name())
			break
		}
	}
	if zipPath == "" {
		return nil, nil, fmt.Errorf("no zip file found in backup")
	}
	content, err := os.ReadFile(zipPath) // #nosec G304 (CWE-22): Potential file inclusion via variable; is okay, since path is generated internally
	if err != nil {
		return nil, nil, err
	}

	volumes, err := extractVolumesFromZipsDockerComposeYaml(content)
	if err != nil {
		return nil, nil, err
	}

	return content, volumes, nil
}

func (b *RealBackupManager) cleanupAndRestoreVolumes(backupId string, volumes, envs []string) error {
	for _, volume := range volumes {
		stopCmd := fmt.Sprintf(
			"docker ps -a --filter \"volume=%s\" --format \"{{.ID}}\" | xargs -r docker rm -f",
			volume,
		)
		if err := runCommand(stopCmd); err != nil {
			Logger.Warn("could not stop container using volume %s: %v", volume, err)
		}
		rmCmd := fmt.Sprintf("docker volume rm %s", volume)
		if err := runCommand(rmCmd); err != nil {
			Logger.Warn("could not delete volume %s: %v", volume, err)
		}
	}

	_, err := executeInResticContainer(
		"restic restore "+backupId+" --target /",
		volumes, nil, envs, "",
	)
	return err
}

func (b *RealBackupManager) finalizeRestore(backupId string, zipFileContent []byte, envs []string) (*tools.RestoredVersionInfo, error) {
	output, err := executeInResticContainer(
		"restic snapshots --json "+backupId,
		nil, nil, envs, "",
	)
	if err != nil {
		return nil, err
	}

	backupInfos, err := parseBackupInfo(output)
	if err != nil {
		return nil, err
	}
	if len(backupInfos) != 1 {
		return nil, fmt.Errorf("expected exactly one backup info, got %d", len(backupInfos))
	}
	info := backupInfos[0]

	restoredVersionInfo := &tools.RestoredVersionInfo{
		Maintainer:     info.Maintainer,
		AppName:        info.AppName,
		VersionName:    info.VersionName,
		VersionContent: zipFileContent,
	}

	app := tools.RepoApp{
		Maintainer:               info.Maintainer,
		AppName:                  info.AppName,
		VersionName:              info.VersionName,
		VersionCreationTimestamp: info.VersionCreationTimestamp,
		VersionContent:           zipFileContent,
		ShouldBeRunning:          true,
	}

	if common.IsOcelotDbApp(app) {
		common.InitializeDatabase(tools.Config.IsUsingDockerNetwork, tools.Config.UseProductionDatabaseContainer)
		if err := common.UpsertApp(app); err != nil {
			return nil, err
		}
	} else {
		if err := common.UpsertApp(app); err != nil {
			return nil, err
		}

		appId, err := common.AppRepo.GetAppId(info.Maintainer, info.AppName)
		if err != nil {
			return nil, err
		}

		if err := clients.Apps.StartApp(appId); err != nil {
			return nil, err
		}
	}

	return restoredVersionInfo, nil
}

func runCommand(command string) error {
	_, err := runCommandWithOutputString(command)
	return err
}

func runCommandWithOutputString(command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	var outputBuffer bytes.Buffer
	if showCommandOutput {
		cmd.Stdout = io.MultiWriter(os.Stdout, &outputBuffer)
		cmd.Stderr = io.MultiWriter(os.Stdout, &outputBuffer)
	} else {
		cmd.Stdout = &outputBuffer
		cmd.Stderr = &outputBuffer
	}
	err := cmd.Run()
	return outputBuffer.String(), err
}

func (r *RealBackupManager) ListAppsInBackupRepo(isLocalBackup bool) ([]tools.MaintainerAndApp, error) {
	allBackups, err := r.getBackupsMatchingTags(isLocalBackup, nil)
	if err != nil {
		return nil, err
	}
	var maintainersAndApps []tools.MaintainerAndApp
	for _, backup := range allBackups {
		maintainersAndApps = append(maintainersAndApps, tools.MaintainerAndApp{
			Maintainer: backup.Maintainer,
			AppName:    backup.AppName,
		})
	}

	return tools.FindUniqueMaintainerAndAppNamePairs(maintainersAndApps), err
}

func (r *RealBackupManager) RunRetentionPolicy() error {
	err := r.applyRetentionPolicyToRepo(true)
	if err != nil {
		return err
	}

	repository, err := ssh.GetRemoteBackupRepository()
	if err != nil {
		return err
	}
	if repository.IsEnabled {
		err = r.applyRetentionPolicyToRepo(false)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *RealBackupManager) applyRetentionPolicyToRepo(isLocalRepo bool) error {
	repoApps, err := clients.BackupManager.ListAppsInBackupRepo(isLocalRepo)
	if err != nil {
		return err
	}
	err = r.runRetentionPolicyOfAppInRepo(repoApps, isLocalRepo)
	if err != nil {
		return err
	}
	return nil
}

func (r *RealBackupManager) runRetentionPolicyOfAppInRepo(apps []tools.MaintainerAndApp, isLocal bool) error {
	for _, app := range apps {
		backups, err := r.ListBackupsOfApp(tools.BackupListRequest{
			Maintainer: app.Maintainer,
			AppName:    app.AppName,
			IsLocal:    isLocal,
		})
		if err != nil {
			Logger.Error("Error running retention policy of app %s: %v", app.AppName, err)
			continue
		}

		backupsToDelete := FindBackupsForDeletionAccordingToRetentionPolicy(backups, numberOfDailyBackupsToKeep, numberOfWeeklyBackupsToKeep, numberOfMonthlyBackupsToKeep)
		for _, backup := range backupsToDelete {
			if backup.Description == tools.ManualBackupDescription {
				continue
			}
			err = r.DeleteBackup(backup.BackupId, isLocal)
			if err != nil {
				Logger.Error("Error deleting backup %s: %v", backup.BackupId, err)
				continue
			}
		}
	}
	return nil
}
