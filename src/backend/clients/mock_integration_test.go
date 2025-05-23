//go:build fast

package clients

import (
	"github.com/ocelot-cloud/shared/assert"
	"net/http/httptest"
	"ocelot/backend/apps/common"
	"ocelot/backend/security"
	"ocelot/backend/ssh"
	"ocelot/backend/tools"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	common.InitializeDatabase(false, false)
	cleanup()
	defer cleanup()
	Apps = &MockAppManager{}
	BackupManager = &MockBackupManager{}
	m.Run()
}

func TestStartingMakeAppAvailable(t *testing.T) {
	defer cleanup()
	assert.Nil(t, common.AppRepo.CreateApp(common.GetSampleAppInfo()))
	rr := httptest.NewRecorder()
	Apps.ProxyRequestToTheAppsDockerContainer(rr, nil)
	assert.Equal(t, "app not available\n", rr.Body.String())
	assert.Equal(t, 400, rr.Code)
	appId, err := common.AppRepo.GetAppId(tools.SampleMaintainer, tools.SampleApp)
	assert.Nil(t, err)
	app, err := common.AppRepo.GetApp(appId)
	assert.Nil(t, err)
	assert.False(t, app.ShouldBeRunning)

	assert.Nil(t, Apps.StartApp(appId))
	rr = httptest.NewRecorder()
	Apps.ProxyRequestToTheAppsDockerContainer(rr, nil)
	assert.Equal(t, "this is version 1.0", rr.Body.String())
	assert.Equal(t, 200, rr.Code)
	app, err = common.AppRepo.GetApp(appId)
	assert.Nil(t, err)
	assert.True(t, app.ShouldBeRunning)

	assert.Nil(t, Apps.StopApp(appId))
	rr = httptest.NewRecorder()
	Apps.ProxyRequestToTheAppsDockerContainer(rr, nil)
	assert.Equal(t, "app not available\n", rr.Body.String())
	assert.Equal(t, 400, rr.Code)
	app, err = common.AppRepo.GetApp(appId)
	assert.Nil(t, err)
	assert.False(t, app.ShouldBeRunning)
}

func TestSampleAppBackupCreationAndRestoration(t *testing.T) {
	defer cleanup()
	assert.Nil(t, common.AppRepo.CreateApp(common.GetSampleAppInfo()))
	appId, err := common.AppRepo.GetAppId(tools.SampleMaintainer, tools.SampleApp)
	assert.Nil(t, err)
	backups, err := BackupManager.ListBackupsOfApp(tools.SampleAppBackupListRequestLocal)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(backups))
	assert.Nil(t, BackupManager.CreateBackup(appId, tools.SampleBackupDescription))

	assertAppRunning(t, appId, false)
	assert.Nil(t, common.AppRepo.SetAppShouldBeRunning(appId, true))
	assertAppRunning(t, appId, true)

	backups, err = BackupManager.ListBackupsOfApp(tools.SampleAppBackupListRequestLocal)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(backups))
	backup := backups[0]
	assert.Equal(t, tools.SampleMaintainer, backup.Maintainer)
	assert.Equal(t, tools.SampleApp, backup.AppName)
	assert.Equal(t, tools.SampleAppVersion1Name, backup.VersionName)
	assert.Equal(t, tools.SampleBackupDescription, backup.Description)
	assert.Equal(t, tools.SampleAppVersion1CreationTimestamp, backup.VersionCreationTimestamp)
	assert.True(t, time.Now().UTC().Add(-1*time.Second).Before(backup.BackupCreationTimestamp))
	assert.True(t, time.Now().UTC().Add(+1*time.Second).After(backup.BackupCreationTimestamp))

	restoredBackupInfo, err := BackupManager.RestoreBackup(tools.BackupOperationRequest{
		BackupId: backup.BackupId,
		IsLocal:  true,
	})
	assert.Nil(t, err)
	assert.Equal(t, tools.SampleMaintainer, restoredBackupInfo.Maintainer)
	assert.Equal(t, tools.SampleApp, restoredBackupInfo.AppName)
	assert.Equal(t, tools.SampleAppVersion1Name, restoredBackupInfo.VersionName)
	assert.Equal(t, tools.GetSampleAppContent(), restoredBackupInfo.VersionContent)

	appId, err = common.AppRepo.GetAppId(tools.SampleMaintainer, tools.SampleApp)
	assert.Nil(t, err)
	assertAppRunning(t, appId, true)

	assert.Nil(t, BackupManager.DeleteBackup(backup.BackupId, true))
	backups, err = BackupManager.ListBackupsOfApp(tools.SampleAppBackupListRequestLocal)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(backups))
}

func TestPostgresBackupCreationAndRestoration(t *testing.T) {
	defer cleanup()
	postgresAppId, err := common.AppRepo.GetAppId(tools.OcelotDbMaintainer, tools.OcelotDbAppName)
	assert.Nil(t, err)
	backups, err := BackupManager.ListBackupsOfApp(tools.OcelotDbAppBackupListRequestLocal)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(backups))

	assert.Nil(t, security.UserRepo.CreateUser("sampleuser", "samplepassword", false))
	users, err := security.UserRepo.ListUsers()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(users))
	assert.Nil(t, BackupManager.CreateBackup(postgresAppId, tools.SampleBackupDescription))

	backups, err = BackupManager.ListBackupsOfApp(tools.OcelotDbAppBackupListRequestLocal)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(backups))
	backup := backups[0]
	assert.Equal(t, tools.OcelotDbMaintainer, backup.Maintainer)
	assert.Equal(t, tools.OcelotDbAppName, backup.AppName)
	assert.Equal(t, common.PostgresVersion, backup.VersionName)
	assert.Equal(t, tools.SampleBackupDescription, backup.Description)
	assert.Equal(t, common.SamplePostgresCreationDate, backup.VersionCreationTimestamp)
	assert.True(t, time.Now().UTC().Add(-1*time.Second).Before(backup.BackupCreationTimestamp))
	assert.True(t, time.Now().UTC().Add(+1*time.Second).After(backup.BackupCreationTimestamp))

	restoredBackupInfo, err := BackupManager.RestoreBackup(tools.BackupOperationRequest{
		BackupId: backup.BackupId,
		IsLocal:  true,
	})
	assert.Nil(t, err)
	assert.Equal(t, tools.OcelotDbMaintainer, restoredBackupInfo.Maintainer)
	assert.Equal(t, tools.OcelotDbAppName, restoredBackupInfo.AppName)
	assert.Equal(t, common.PostgresVersion, restoredBackupInfo.VersionName)

	assert.Nil(t, BackupManager.DeleteBackup(backup.BackupId, true))
	_, err = common.AppRepo.GetAppId(tools.OcelotDbMaintainer, tools.OcelotDbAppName)
	assert.Nil(t, err)

	ocelotDbAppId, err := common.AppRepo.GetAppId(tools.OcelotDbMaintainer, tools.OcelotDbAppName)
	assert.Nil(t, err)
	assertAppRunning(t, ocelotDbAppId, true)
	backups, err = BackupManager.ListBackupsOfApp(tools.OcelotDbAppBackupListRequestLocal)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(backups))
}

func assertAppRunning(t *testing.T, appId int, shouldBeRunning bool) {
	app, err := common.AppRepo.GetApp(appId)
	assert.Nil(t, err)
	assert.Equal(t, shouldBeRunning, app.ShouldBeRunning)
}

func TestUpdate(t *testing.T) {
	defer cleanup()
	assert.Nil(t, common.AppRepo.CreateApp(common.GetSampleAppInfo()))
	appId, err := common.AppRepo.GetAppId(tools.SampleMaintainer, tools.SampleApp)

	assert.Nil(t, err)
	backups, err := BackupManager.ListBackupsOfApp(tools.SampleAppBackupListRequestLocal)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(backups))
	app, err := common.AppRepo.GetApp(appId)
	assert.Nil(t, err)
	assert.Equal(t, tools.SampleAppVersion1Name, app.VersionName)

	assert.Nil(t, BackupManager.UpdateAppVersion(appId))
	backups, err = BackupManager.ListBackupsOfApp(tools.SampleAppBackupListRequestLocal)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(backups))
	backup := backups[0]
	assert.Equal(t, tools.SampleAppVersion1Name, backup.VersionName)
	app, err = common.AppRepo.GetApp(appId)
	assert.Nil(t, err)
	assert.Equal(t, tools.SampleAppVersion2Name, app.VersionName)
}

func TestPruneDoesNotDeleteBackups(t *testing.T) {
	defer cleanup()
	assert.Nil(t, common.AppRepo.CreateApp(common.GetSampleAppInfo()))
	appId, err := common.AppRepo.GetAppId(tools.SampleMaintainer, tools.SampleApp)
	assert.Nil(t, err)
	assert.Nil(t, BackupManager.CreateBackup(appId, tools.ManualBackupDescription))
	backups, err := BackupManager.ListBackupsOfApp(tools.SampleAppBackupListRequestLocal)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(backups))

	assert.Nil(t, BackupManager.PruneApp(appId))

	_, err = common.AppRepo.GetAppId(tools.SampleMaintainer, tools.SampleApp)
	assert.NotNil(t, err)
	assert.Nil(t, common.AppRepo.CreateApp(common.GetSampleAppInfo()))
	_, err = common.AppRepo.GetAppId(tools.SampleMaintainer, tools.SampleApp)
	assert.Nil(t, err)
	backups, err = BackupManager.ListBackupsOfApp(tools.SampleAppBackupListRequestLocal)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(backups))
}

func TestListBackups(t *testing.T) {
	defer cleanup()
	assert.Nil(t, common.AppRepo.CreateApp(common.GetSampleAppInfo()))
	appId, err := common.AppRepo.GetAppId(tools.SampleMaintainer, tools.SampleApp)
	assert.Nil(t, err)
	err = BackupManager.CreateBackup(appId, tools.ManualBackupDescription)
	assert.Nil(t, err)

	backups, err := BackupManager.ListBackupsOfApp(tools.SampleAppBackupListRequestLocal)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(backups))

	backups, err = BackupManager.ListBackupsOfApp(tools.OcelotDbAppBackupListRequestLocal)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(backups))
}

func TestRestorationRecreatesUsersFromBackup(t *testing.T) {
	defer cleanup()
	assert.Nil(t, security.UserRepo.CreateUser("sampleuser", "samplepassword", false))
	postgresAppId, err := common.AppRepo.GetAppId(tools.OcelotDbMaintainer, tools.OcelotDbAppName)
	assert.Nil(t, err)
	err = BackupManager.CreateBackup(postgresAppId, tools.ManualBackupDescription)
	assert.Nil(t, err)

	userId, err := security.UserRepo.GetUserId("sampleuser")
	assert.Nil(t, err)
	assert.Nil(t, security.UserRepo.DeleteUser(userId))

	users, err := security.UserRepo.ListUsers()
	assert.Nil(t, err)
	assert.Equal(t, 0, len(users))

	backups, err := BackupManager.ListBackupsOfApp(tools.OcelotDbAppBackupListRequestLocal)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(backups))
	backup := backups[0]

	_, err = BackupManager.RestoreBackup(tools.BackupOperationRequest{
		BackupId: backup.BackupId,
		IsLocal:  true,
	})
	assert.Nil(t, err)

	users, err = security.UserRepo.ListUsers()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(users))
	assert.Equal(t, "sampleuser", users[0].Name)
	assert.False(t, users[0].IsAdmin)
}

func TestLocalVsRemoteBackup(t *testing.T) {
	defer cleanup()
	assert.Nil(t, common.AppRepo.CreateApp(common.GetSampleAppInfo()))
	appId, err := common.AppRepo.GetAppId(tools.SampleMaintainer, tools.SampleApp)
	assert.Nil(t, err)

	assertNumberOfBackups(t, tools.SampleAppBackupListRequestRemote, 0, 0)
	assert.Nil(t, BackupManager.CreateBackup(appId, tools.ManualBackupDescription))
	assertNumberOfBackups(t, tools.SampleAppBackupListRequestRemote, 1, 0)

	enableRemoteBackupRepo(t)
	assert.Nil(t, BackupManager.CreateBackup(appId, tools.ManualBackupDescription))

	assertNumberOfBackups(t, tools.SampleAppBackupListRequestRemote, 2, 1)
	remoteBackups, err := BackupManager.ListBackupsOfApp(tools.SampleAppBackupListRequestRemote)
	assert.Nil(t, err)
	remoteBackup := remoteBackups[0]
	assert.Nil(t, BackupManager.DeleteBackup(remoteBackup.BackupId, false))
	assertNumberOfBackups(t, tools.SampleAppBackupListRequestRemote, 2, 0)
}

func enableRemoteBackupRepo(t *testing.T) {
	err := ssh.SetRemoteBackupRepository(tools.RemoteBackupRepository{
		IsEnabled:          true,
		Host:               "",
		SshPort:            "",
		SshUser:            "",
		SshPassword:        "",
		EncryptionPassword: "",
	})
	assert.Nil(t, err)
}

func assertNumberOfBackups(t *testing.T, sampleAppBackupRequest tools.BackupListRequest, expectedLocalNumber, expectedRemoteNumber int) {
	sampleAppBackupRequest.IsLocal = true
	backups, err := BackupManager.ListBackupsOfApp(sampleAppBackupRequest)
	assert.Nil(t, err)
	assert.Equal(t, expectedLocalNumber, len(backups))

	sampleAppBackupRequest.IsLocal = false
	backups, err = BackupManager.ListBackupsOfApp(sampleAppBackupRequest)
	assert.Nil(t, err)
	assert.Equal(t, expectedRemoteNumber, len(backups))
}

func cleanup() {
	common.WipeWholeDatabase()
	mockMgr, ok := BackupManager.(*MockBackupManager)
	if ok {
		mockMgr.Backups = nil
	}
}

func TestListAppsInBackupRepoMock(t *testing.T) {
	defer cleanup()
	assert.Nil(t, common.AppRepo.CreateApp(common.GetSampleAppInfo()))
	appId, err := common.AppRepo.GetAppId(tools.SampleMaintainer, tools.SampleApp)
	assert.Nil(t, err)

	repo, err := BackupManager.ListAppsInBackupRepo(true)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(repo))

	assert.Nil(t, BackupManager.CreateBackup(appId, tools.ManualBackupDescription))
	repo, err = BackupManager.ListAppsInBackupRepo(true)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(repo))
	assert.Equal(t, repo[0].Maintainer, tools.SampleMaintainer)
	assert.Equal(t, repo[0].AppName, tools.SampleApp)
	repo, err = BackupManager.ListAppsInBackupRepo(false)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(repo))

	enableRemoteBackupRepo(t)

	secondApp := common.GetSampleAppInfo()
	secondApp.AppName = tools.SampleApp + "2"
	assert.Nil(t, common.AppRepo.CreateApp(secondApp))
	app2Id, err := common.AppRepo.GetAppId(tools.SampleMaintainer, tools.SampleApp+"2")
	assert.Nil(t, err)

	assert.Nil(t, BackupManager.CreateBackup(app2Id, tools.ManualBackupDescription))

	repo, err = BackupManager.ListAppsInBackupRepo(true)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(repo))
	assert.Equal(t, repo[0].Maintainer, tools.SampleMaintainer)
	assert.Equal(t, repo[0].AppName, tools.SampleApp)
	assert.Equal(t, repo[1].Maintainer, tools.SampleMaintainer)
	assert.Equal(t, repo[1].AppName, tools.SampleApp+"2")
	repo, err = BackupManager.ListAppsInBackupRepo(false)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(repo))
	assert.Equal(t, repo[0].Maintainer, tools.SampleMaintainer)
	assert.Equal(t, repo[0].AppName, tools.SampleApp+"2")
}

func TestBackupListContainOnlyAppSpecificBackups(t *testing.T) {
	defer cleanup()
	app1 := common.GetSampleAppInfo()
	assert.Nil(t, common.AppRepo.CreateApp(app1))
	appId, err := common.AppRepo.GetAppId(tools.SampleMaintainer, tools.SampleApp)
	assert.Nil(t, err)

	assert.Nil(t, BackupManager.CreateBackup(appId, tools.ManualBackupDescription))
	apps, err := BackupManager.ListBackupsOfApp(tools.BackupListRequest{Maintainer: app1.Maintainer, AppName: app1.AppName, IsLocal: true})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(apps))

	app2 := common.GetSampleAppInfo()
	app2.AppName = tools.SampleApp + "2"
	assert.Nil(t, common.AppRepo.CreateApp(app2))
	app2Id, err := common.AppRepo.GetAppId(tools.SampleMaintainer, tools.SampleApp+"2")
	assert.Nil(t, err)
	assert.Nil(t, BackupManager.CreateBackup(app2Id, tools.ManualBackupDescription))
	apps2, err := BackupManager.ListBackupsOfApp(tools.BackupListRequest{Maintainer: app2.Maintainer, AppName: app2.AppName, IsLocal: true})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(apps))
	assert.Equal(t, 1, len(apps2))
}
