package backups

import (
	"fmt"
	"github.com/ocelot-cloud/shared/utils"
	"ocelot/backend/apps/common"
	"ocelot/backend/clients"
	"ocelot/backend/security"
	"ocelot/backend/tools"
	"strings"
)

var (
	fileBackupTestAppImageName        = "app:local"
	fileBackupTestAppDockerfile       = "Dockerfile.file_backup_test_app"
	createCreateSampleappImageCommand = fmt.Sprintf("docker build -t %s -f %s/%s .", fileBackupTestAppImageName, tools.DockerDir, fileBackupTestAppDockerfile)

	resticImageName         = "restic:local"
	createCreateResticImage = fmt.Sprintf("docker build -t %s -f %s/Dockerfile.restic .", resticImageName, tools.DockerDir)
)

var Logger = tools.Logger

func ProvideBackupClient() clients.BackupManagerInterface {
	if tools.AreMocksUsed() {
		return &clients.MockBackupManager{}
	} else {
		err := PrepareLocalBackupContainer()
		if err != nil {
			Logger.Fatal("Error preparing backup container and repos: %v", err)
		}
		return &RealBackupManager{}
	}
}

func InitializeBackupsModule() {
	routes := []security.Route{
		{Path: tools.BackupsCreatePath, HandlerFunc: CreateBackupHandler, AccessLevel: security.Admin},
		{Path: tools.BackupsListPath, HandlerFunc: ListBackupsHandler, AccessLevel: security.Admin},
		{Path: tools.BackupsRestorePath, HandlerFunc: RestoreBackupHandler, AccessLevel: security.Admin},
		{Path: tools.BackupsDeletePath, HandlerFunc: DeleteBackupHandler, AccessLevel: security.Admin},
		{Path: tools.BackupsListAppsPath, HandlerFunc: ListAppsOfBackupRepository, AccessLevel: security.Admin},

		{Path: tools.SettingsMaintenanceReadPath, HandlerFunc: GetMaintenanceSettingsHandler, AccessLevel: security.Admin},
		{Path: tools.SettingsMaintenanceSavePath, HandlerFunc: SetMaintenanceSettingsHandler, AccessLevel: security.Admin},
	}
	security.RegisterRoutes(routes)
}

func PrepareLocalBackupContainer() error {
	err := tools.AppOperationMutex.TryLock("prepare local backup container")
	if err != nil {
		return err
	}
	defer tools.AppOperationMutex.Unlock()

	checkImage := "docker images -q restic:local"
	output, err := runCommandWithOutputString(checkImage)
	if err != nil {
		Logger.Error("could not check for restic image: %v", err)
		return err
	}

	isResticImageMissing := strings.TrimSpace(output) == ""
	if isResticImageMissing {
		Logger.Info("restic image is missing, creating it")
		err = runCommand(createCreateResticImage)
		if err != nil {
			Logger.Fatal("could not create restic image: %v", err)
		}
		Logger.Info("restic image was created successfully")
	} else {
		Logger.Info("restic image already exists, skipping creation")
	}
	return err
}

func WipeDatabaseForTesting() {
	_, err := common.DB.Exec("DELETE FROM configs")
	if err != nil {
		Logger.Fatal("Database wipe failed: %v", err)
	}

	_, err = common.DB.Exec("DELETE FROM users WHERE NOT user_name = 'admin'")
	if err != nil {
		Logger.Fatal("Database wipe failed: %v", err)
	}

	apps, _ := common.AppRepo.ListApps()
	for _, app := range apps {
		if !common.IsOcelotDbApp(app) && clients.BackupManager != nil {
			err = clients.BackupManager.PruneApp(app.AppId)
			if err != nil {
				Logger.Error("Prune app failed: %v", err)
			}
		}
	}

	err = utils.ExecuteShellCommand("docker rm -rf samplemaintainer_sampleapp_sampleapp")
	if err != nil {
		Logger.Info("no sample app container to remove")
	}
	err = utils.ExecuteShellCommand("docker volume rm backups")
	if err != nil {
		Logger.Info("no backup volume to remove")
	}

	backupManager, ok := clients.BackupManager.(*clients.MockBackupManager)
	if ok {
		backupManager.Backups = nil
	}

	SetLastMaintenanceCycleExecutionDate(UnixEpochStartTime)
	err = SetDefaultMaintenanceSettingsIfNotExisting()
	if err != nil {
		Logger.Error("SetDefaultMaintenanceSettings failed: %v", err)
	}
}
