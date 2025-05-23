package backups

import (
	"errors"
	"fmt"
	"ocelot/backend/apps/cloud"
	"ocelot/backend/apps/common"
	"ocelot/backend/clients"
	"ocelot/backend/tools"
	"strconv"
)

func (b *RealBackupManager) UpdateAppVersion(appId int) error {
	defer cloud.UpdateAppConfigs()
	err := clients.Apps.StopApp(appId)
	if err != nil {
		return err
	}

	app, err := common.AppRepo.GetApp(appId)
	if err != nil {
		return err
	}

	versions, err := common.StoreClient.GetVersions(strconv.Itoa(appId))
	if err != nil {
		return err
	}
	if len(versions) == 0 {
		return fmt.Errorf("no versions found for app")
	}
	latestVersionInAppStore := getLatestVersion(versions)

	if latestVersionInAppStore.VersionCreationTimestamp.After(app.VersionCreationTimestamp) {
		Logger.Info("starting update of app %s", app.AppName)
		err = clients.BackupManager.CreateBackup(appId, tools.AutoBackupDescription)
		if err != nil {
			return err
		}
		downloadedRepoApp, err := common.DownloadTag(latestVersionInAppStore.Id)
		if err != nil {
			return err
		}
		err = common.UpsertApp(*downloadedRepoApp)
		if err != nil {
			Logger.Error("failed to upsert latest version: %v. Trying to recover old app.", err)
			err = common.UpsertApp(*app)
			if err != nil {
				Logger.Error("failed to recover old app: %v", err)
				return err
			}
			return err
		}
		err = clients.Apps.StartApp(appId)
		if err != nil {
			return err
		}
	} else {
		msg := clients.GetUpdateErrorString(*app)
		Logger.Info(msg)
		return errors.New(msg)
	}

	return nil
}

func getLatestVersion(versions []tools.VersionInfo) tools.VersionInfo {
	latest := versions[0]
	for _, version := range versions {
		if version.VersionCreationTimestamp.After(latest.VersionCreationTimestamp) {
			latest = version
		}
	}
	return latest
}
