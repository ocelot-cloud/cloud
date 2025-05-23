package clients

import (
	"errors"
	"fmt"
	"ocelot/backend/apps/common"
	"ocelot/backend/security"
	"ocelot/backend/ssh"
	"ocelot/backend/tools"
	"time"
)

var (
	BackupManager BackupManagerInterface
)

type BackupManagerInterface interface {
	CreateBackup(appId int, description tools.BackupDescription) error
	DeleteBackup(backupId string, isLocalBackup bool) error
	RestoreBackup(backupRestoreRequest tools.BackupOperationRequest) (*tools.RestoredVersionInfo, error)
	UpdateAppVersion(appId int) error
	PruneApp(appId int) error

	ListBackupsOfApp(request tools.BackupListRequest) ([]tools.BackupInfo, error)
	ListAppsInBackupRepo(isLocalBackup bool) ([]tools.MaintainerAndApp, error)
	RunRetentionPolicy() error
}

type backupFullInfo struct {
	backupInfo     tools.BackupInfo
	versionContent []byte
	users          []tools.UserFullInfo
	isLocal        bool
}

type MockBackupManager struct {
	Backups []backupFullInfo
}

var backupIdSource = 100

func (m *MockBackupManager) UpdateAppVersion(appId int) error {
	app, err := common.AppRepo.GetApp(appId)
	if err != nil {
		return err
	}
	if app.Maintainer != tools.SampleMaintainer || app.AppName != tools.SampleApp {
		return fmt.Errorf("mock can only update sample app")
	}
	if app.VersionName == tools.SampleAppVersion1Name {
		err = BackupManager.CreateBackup(appId, tools.AutoBackupDescription)
		if err != nil {
			return err
		}
		versionMetaData := tools.VersionMetaData{
			Name:              tools.SampleAppVersion2Name,
			CreationTimestamp: time.Now().UTC(),
			Content:           tools.GetSampleAppContent(),
		}
		err = common.AppRepo.UpdateVersion(appId, versionMetaData)
		if err != nil {
			return err
		}
		return nil
	} else {
		msg := GetUpdateErrorString(*app)
		return errors.New(msg)
	}
}

func GetUpdateErrorString(app tools.RepoApp) string {
	return fmt.Sprintf("can't update app '%s / %s' because the latest version '%s' is already installed", app.Maintainer, app.AppName, app.VersionName)
}

func (m *MockBackupManager) PruneApp(appId int) error {
	return common.AppRepo.DeleteApp(appId)
}

func (m *MockBackupManager) CreateBackup(appId int, description tools.BackupDescription) error {
	app, err := common.AppRepo.GetApp(appId)
	if err != nil {
		return err
	}
	newBackup := tools.BackupInfo{
		BackupId:                 fmt.Sprintf("%064d", backupIdSource), // must always have 64 digits for input validation
		Maintainer:               app.Maintainer,
		AppName:                  app.AppName,
		VersionName:              app.VersionName,
		VersionCreationTimestamp: app.VersionCreationTimestamp,
		Description:              description,
		BackupCreationTimestamp:  time.Now().UTC(),
	}
	backupIdSource++
	backup := backupFullInfo{
		backupInfo:     newBackup,
		versionContent: app.VersionContent,
		isLocal:        true,
	}

	if common.IsOcelotDbApp(*app) {
		users, err := security.UserRepo.GetAllUsersFullInfo()
		if err != nil {
			return err
		}
		backup.users = users
	}

	m.Backups = append(m.Backups, backup)
	isRemoteBackupEnabled, err := ssh.IsRemoteBackupEnabled()
	if err != nil {
		return err
	} else if isRemoteBackupEnabled {
		backup.isLocal = false
		m.Backups = append(m.Backups, backup)
	}
	return err
}

func (m *MockBackupManager) ListBackupsOfApp(backupListRequest tools.BackupListRequest) ([]tools.BackupInfo, error) {
	var appBackups []tools.BackupInfo
	for _, backup := range m.Backups {
		if backup.backupInfo.Maintainer == backupListRequest.Maintainer && backup.backupInfo.AppName == backupListRequest.AppName && backup.isLocal == backupListRequest.IsLocal {
			appBackups = append(appBackups, backup.backupInfo)
		}
	}
	return appBackups, nil
}

func (m *MockBackupManager) DeleteBackup(backupId string, isLocalBackup bool) error {
	var wasBackupFound = false
	for i, backup := range m.Backups {
		if backup.backupInfo.BackupId == backupId && backup.isLocal == isLocalBackup {
			wasBackupFound = true
			m.Backups = append(m.Backups[:i], m.Backups[i+1:]...)
		}
	}
	if wasBackupFound {
		return nil
	} else {
		return fmt.Errorf("backup not found")
	}
}

func (m *MockBackupManager) RestoreBackup(request tools.BackupOperationRequest) (*tools.RestoredVersionInfo, error) {
	var backup *backupFullInfo
	for _, currentBackup := range m.Backups {
		if currentBackup.backupInfo.BackupId == request.BackupId && currentBackup.isLocal == request.IsLocal {
			backup = &currentBackup
		}
	}

	if backup == nil {
		return nil, fmt.Errorf("backup id does not exist")
	}

	restoredVersionInfo := &tools.RestoredVersionInfo{
		Maintainer:     backup.backupInfo.Maintainer,
		AppName:        backup.backupInfo.AppName,
		VersionName:    backup.backupInfo.VersionName,
		VersionContent: backup.versionContent,
	}

	existingAppId, err := common.AppRepo.GetAppId(restoredVersionInfo.Maintainer, restoredVersionInfo.AppName)
	if err == nil {
		err = common.AppRepo.DeleteApp(existingAppId)
		if err != nil {
			return nil, err
		}
	}

	newRepoApp := tools.RepoApp{
		AppId:                    0,
		Maintainer:               restoredVersionInfo.Maintainer,
		AppName:                  restoredVersionInfo.AppName,
		VersionName:              restoredVersionInfo.VersionName,
		VersionCreationTimestamp: backup.backupInfo.VersionCreationTimestamp,
		VersionContent:           restoredVersionInfo.VersionContent,
		ShouldBeRunning:          true,
	}
	err = common.AppRepo.CreateApp(newRepoApp)
	if err != nil {
		return nil, err
	}

	err = security.UserRepo.DeleteUsersAndAddUsersFullInfo(backup.users)
	if err != nil {
		return nil, err
	}

	return restoredVersionInfo, nil
}

func (m *MockBackupManager) ListAppsInBackupRepo(isLocalBackup bool) ([]tools.MaintainerAndApp, error) {
	var allFoundApps []tools.MaintainerAndApp
	for _, backup := range m.Backups {
		if backup.isLocal == isLocalBackup {
			allFoundApps = append(allFoundApps, tools.MaintainerAndApp{
				Maintainer: backup.backupInfo.Maintainer,
				AppName:    backup.backupInfo.AppName,
			})
		}
	}
	return tools.FindUniqueMaintainerAndAppNamePairs(allFoundApps), nil
}

func (m *MockBackupManager) RunRetentionPolicy() error {
	// only needed for real backup manager
	return nil
}
