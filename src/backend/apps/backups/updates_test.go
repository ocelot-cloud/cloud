//go:build slow

package backups

import (
	"fmt"
	"github.com/ocelot-cloud/shared/assert"
	"github.com/ocelot-cloud/shared/utils"
	"io"
	"net/http"
	"ocelot/backend/apps/common"
	"ocelot/backend/apps/store"
	"ocelot/backend/clients"
	"ocelot/backend/tools"
	"strings"
	"testing"
	"time"
)

func TestUpdateAppVersion(t *testing.T) {
	defer updateUnitCleanup(t)
	updateUnitSetup(t)
	appId := prepareAppWithOldVersion(t)
	appBackup := performUpdate(t, appId)
	removeSampleApp(t, appId)
	restoredVersionInfo, err := clients.BackupManager.RestoreBackup(tools.BackupOperationRequest{
		BackupId: appBackup.BackupId,
		IsLocal:  true,
	})
	assert.Nil(t, err)
	assertRestoredBackup(t, restoredVersionInfo)
}

func performUpdate(t *testing.T, appId int) tools.BackupInfo {
	appBackups, err := clients.BackupManager.ListBackupsOfApp(tools.SampleAppBackupListRequestLocal)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(appBackups))

	assert.Nil(t, clients.BackupManager.UpdateAppVersion(appId))

	app, err := common.AppRepo.GetApp(appId)
	assert.Nil(t, err)
	_, err = common.AppRepo.GetApp(appId)
	assert.Nil(t, err)

	assert.Equal(t, tools.SampleAppVersion2Name, app.VersionName)

	responseText, err := getContentOfIndexPage()
	assert.Nil(t, err)
	assert.True(t, strings.Contains(responseText, "this is version 2.0"))

	appBackups, err = clients.BackupManager.ListBackupsOfApp(tools.SampleAppBackupListRequestLocal)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(appBackups))
	appBackup := appBackups[0]
	assert.Equal(t, tools.SampleMaintainer, appBackup.Maintainer)
	assert.Equal(t, tools.SampleApp, appBackup.AppName)
	assert.Equal(t, tools.SampleAppVersion1Name, appBackup.VersionName)
	assert.Equal(t, tools.SampleAppVersion1CreationTimestamp, appBackup.VersionCreationTimestamp)
	assert.Equal(t, tools.AutoBackupDescription, appBackup.Description)
	return appBackup
}

func removeSampleApp(t *testing.T, appId int) {
	shouldSampleAppVolumeBePresent(t, true)
	removeAppContainerIfPresent(t)
	removeAppVolume(t)
	shouldSampleAppVolumeBePresent(t, false)
	assert.Nil(t, common.AppRepo.DeleteApp(appId))
	apps, err := common.AppRepo.ListApps()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(apps))
}

func prepareAppWithOldVersion(t *testing.T) int {
	downloadedRepoApp, err := common.DownloadTag(tools.SampleAppVersion1Id)
	assert.Nil(t, err)
	err = common.UpsertApp(*downloadedRepoApp)
	assert.Nil(t, err)

	appId, err := common.AppRepo.GetAppId(tools.SampleMaintainer, tools.SampleApp)
	assert.Nil(t, err)

	removeAppContainerIfPresent(t)
	assert.Nil(t, clients.Apps.StartApp(appId))
	responseText, err := getContentOfIndexPage()
	assert.Nil(t, err)
	assert.True(t, strings.Contains(responseText, "this is version 1.0"))

	// The upgrade command should stop the container per se, but since I use the "build" keyword, Docker caches the old image. In production I have banned that word, so this is only necessary for testing.
	removeAppContainerIfPresent(t)

	return appId
}

func assertRestoredBackup(t *testing.T, restoredVersionInfo *tools.RestoredVersionInfo) {
	assert.Equal(t, tools.SampleAppVersion1Name, restoredVersionInfo.VersionName)
	assert.Equal(t, tools.GetSampleAppContent(), restoredVersionInfo.VersionContent)
	assert.Equal(t, tools.SampleMaintainer, restoredVersionInfo.Maintainer)
	assert.Equal(t, tools.SampleApp, restoredVersionInfo.AppName)
	shouldSampleAppVolumeBePresent(t, true)

	apps, err := common.AppRepo.ListApps()
	assert.Nil(t, err)
	assert.Equal(t, 2, len(apps))
	appId, err := common.AppRepo.GetAppId(tools.SampleMaintainer, tools.SampleApp)
	assert.Nil(t, err)
	restoredApp, err := common.AppRepo.GetApp(appId)
	assert.Nil(t, err)

	assert.Equal(t, tools.SampleAppVersion1Name, restoredApp.VersionName)
	assert.Equal(t, tools.GetSampleAppContent(), restoredApp.VersionContent)
	assert.Equal(t, tools.SampleAppVersion1CreationTimestamp, restoredApp.VersionCreationTimestamp)
	assert.Equal(t, tools.SampleMaintainer, restoredApp.Maintainer)
	assert.Equal(t, tools.SampleApp, restoredApp.AppName)

	responseText, err := getContentOfIndexPage()
	assert.Nil(t, err)
	assert.True(t, strings.Contains(responseText, "this is version 1.0"))
}

func updateUnitSetup(t *testing.T) {
	common.WipeWholeDatabase()
	err := PrepareLocalBackupContainer()
	assert.Nil(t, err)
	common.StoreClient = store.ProvideAppStoreClient(store.MOCK)
}

func updateUnitCleanup(t *testing.T) {
	common.WipeWholeDatabase()
	removeAppContainerIfPresent(t)
	removeAppVolume(t)
	removeBackupVolume(t)
}

func shouldSampleAppVolumeBePresent(t *testing.T, b bool) {
	var invertResultSymbol string
	if b {
		invertResultSymbol = ""
	} else {
		invertResultSymbol = "! "
	}

	command := invertResultSymbol + "docker volume ls | grep -q " + tools.SampleAppDockerVolume
	err := utils.ExecuteShellCommand(command)
	assert.Nil(t, err)
}

func removeAppVolume(t *testing.T) {
	command := fmt.Sprintf("docker volume rm %s || true", tools.SampleAppDockerVolume)
	err := utils.ExecuteShellCommand(command)
	assert.Nil(t, err)
}

func removeBackupVolume(t *testing.T) {
	err := utils.ExecuteShellCommand("docker volume rm backups || true")
	assert.Nil(t, err)
}

func removeAppContainerIfPresent(t *testing.T) {
	containerRemoveCommand := fmt.Sprintf("docker rm -f %s || true", tools.SampleAppDockerContainer)
	err := utils.ExecuteShellCommand(containerRemoveCommand)
	assert.Nil(t, err)
}

func getContentOfIndexPage() (string, error) {
	const retries = 10

	for i := 0; i < retries; i++ {
		resp, err := http.Get("http://localhost:8085/api")
		if err == nil {
			defer utils.Close(resp.Body)
			body, err := io.ReadAll(resp.Body)
			if err == nil {
				return string(body), nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	return "", fmt.Errorf("failed to get content of index page after 10 attempts")
}
