//go:build slow

package backups

import (
	"github.com/ocelot-cloud/shared/assert"
	"ocelot/backend/apps/common"
	"ocelot/backend/apps/store"
	"ocelot/backend/clients"
	"ocelot/backend/ssh"
	"ocelot/backend/tools"
	"testing"
	"time"
)

func TestBackupsAreNotAppliedWhenAutoBackupOptionsAreDisabled(t *testing.T) {
	defer cleanup()
	app := setupRetentionPolicyTest(t)

	maintenanceSettings, err := GetMaintenanceSettings()
	assert.Nil(t, err)
	maintenanceSettings.AreAutoUpdatesEnabled = false
	maintenanceSettings.AreAutoBackupsEnabled = false
	assert.Nil(t, SetMaintenanceSettings(*maintenanceSettings))

	createBackupsAndConductUpdates(app)
	assertNoBackupsPresent(t, tools.SampleAppBackupListRequestLocal)
}

func setupRetentionPolicyTest(t *testing.T) tools.RepoApp {
	setup()
	common.StoreClient = store.ProvideAppStoreClient(store.MOCK)
	assert.Nil(t, common.CreateSampleAppInRepo())
	assertNoBackupsPresent(t, tools.SampleAppBackupListRequestLocal)
	appId, err := common.AppRepo.GetAppId(tools.SampleMaintainer, tools.SampleApp)
	assert.Nil(t, err)
	app, err := common.AppRepo.GetApp(appId)
	assert.Nil(t, err)
	assert.Nil(t, SetDefaultMaintenanceSettingsIfNotExisting())
	return *app
}

func TestRunRetentionPolicy(t *testing.T) {
	defer cleanup()
	app := setupRetentionPolicyTest(t)

	createBackupsAndConductUpdates(app)
	assert.Nil(t, clients.BackupManager.RunRetentionPolicy())
	backups, err := clients.BackupManager.ListBackupsOfApp(tools.SampleAppBackupListRequestLocal)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(backups))
	assert.Equal(t, tools.AutoBackupDescription, backups[0].Description)
	assert.Equal(t, tools.SampleAppVersion1Name, backups[0].VersionName)

	createBackupsAndConductUpdates(app)
	assert.Nil(t, clients.BackupManager.RunRetentionPolicy())
	backups, err = clients.BackupManager.ListBackupsOfApp(tools.SampleAppBackupListRequestLocal)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(backups))
	assert.Equal(t, tools.AutoBackupDescription, backups[0].Description)
	assert.Equal(t, tools.SampleAppVersion2Name, backups[0].VersionName)
	oldBackupCreationTime := backups[0].BackupCreationTimestamp

	createBackupsAndConductUpdates(app)
	assert.Nil(t, clients.BackupManager.RunRetentionPolicy())
	backups, err = clients.BackupManager.ListBackupsOfApp(tools.SampleAppBackupListRequestLocal)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(backups))
	assert.Equal(t, tools.AutoBackupDescription, backups[0].Description)
	assert.Equal(t, tools.SampleAppVersion2Name, backups[0].VersionName)
	newBackupCreationTime := backups[0].BackupCreationTimestamp
	assert.True(t, newBackupCreationTime.After(oldBackupCreationTime))
}

func TestRetentionPolicyAppliesToEachAppSeparately(t *testing.T) {
	defer cleanup()
	setupRetentionPolicyTest(t)

	appBackups, err := clients.BackupManager.ListBackupsOfApp(tools.SampleAppBackupListRequestLocal)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(appBackups))
	ocelotDbBackups, err := clients.BackupManager.ListBackupsOfApp(tools.OcelotDbAppBackupListRequestLocal)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(ocelotDbBackups))

	maintenanceSettings, err := GetMaintenanceSettings()
	assert.Nil(t, err)
	maintenanceSettings.AreAutoUpdatesEnabled = false
	maintenanceSettings.AreAutoBackupsEnabled = true
	assert.Nil(t, SetMaintenanceSettings(*maintenanceSettings))

	// The extra minute is needed due to inaccurate saving of that time in the database
	timeBeforeRunningRetentionPolicy := time.Now().Add(-1 * time.Minute)
	conductMaintenanceTasks()

	lastExecutionTime, err := getLastMaintenanceCycleExecutionDate()
	assert.Nil(t, err)
	assert.True(t, timeBeforeRunningRetentionPolicy.Before(*lastExecutionTime))
	assert.True(t, lastExecutionTime.Before(time.Now()))

	appBackups, err = clients.BackupManager.ListBackupsOfApp(tools.SampleAppBackupListRequestLocal)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(appBackups))
	ocelotDbBackups, err = clients.BackupManager.ListBackupsOfApp(tools.OcelotDbAppBackupListRequestLocal)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(ocelotDbBackups))
}

func TestManualBackupsAreNotAffectedByRetentionPolicy(t *testing.T) {
	defer cleanup()
	app := setupRetentionPolicyTest(t)

	assertNoBackupsPresent(t, tools.SampleAppBackupListRequestLocal)
	assert.Nil(t, clients.BackupManager.CreateBackup(app.AppId, tools.ManualBackupDescription))
	assert.Nil(t, clients.BackupManager.CreateBackup(app.AppId, tools.ManualBackupDescription))

	backups, err := clients.BackupManager.ListBackupsOfApp(tools.SampleAppBackupListRequestLocal)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(backups))

	assert.Nil(t, clients.BackupManager.RunRetentionPolicy())

	backups, err = clients.BackupManager.ListBackupsOfApp(tools.SampleAppBackupListRequestLocal)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(backups))
}

func TestThatRetentionHandlesAppsFromSameMaintainerSeparately(t *testing.T) {
	defer cleanup()
	setupRetentionPolicyTest(t)
	app2Info := common.GetSampleAppInfo()
	app2Info.AppName += "2"
	assert.Nil(t, common.AppRepo.CreateApp(app2Info))

	maintenanceSettings, err := GetMaintenanceSettings()
	assert.Nil(t, err)
	maintenanceSettings.AreAutoUpdatesEnabled = false
	maintenanceSettings.AreAutoBackupsEnabled = true
	assert.Nil(t, SetMaintenanceSettings(*maintenanceSettings))
	conductMaintenanceTasks()

	app1Backups, err := clients.BackupManager.ListBackupsOfApp(tools.SampleAppBackupListRequestLocal)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(app1Backups))

	app2Request := tools.SampleAppBackupListRequestLocal
	app2Request.AppName = app2Info.AppName
	app2Backups, err := clients.BackupManager.ListBackupsOfApp(app2Request)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(app2Backups))
}

func TestRetentionPolicyForRemoteBackupRepository(t *testing.T) {
	setup()
	defer cleanup()
	defer ssh.ShutDownSshTestContainer()
	ssh.StartSshTestContainer()
	common.InitializeDatabase(false, false)
	defer common.WipeWholeDatabase()

	remoteRepo := ssh.GetSampleRemoteRepo()
	knownHosts, err := ssh.SshClient.GetKnownHosts(remoteRepo.Host, remoteRepo.SshPort)
	assert.Nil(t, err)
	remoteRepo.SshKnownHosts = knownHosts
	assert.Nil(t, ssh.SetRemoteBackupRepository(remoteRepo))

	assert.Nil(t, common.CreateSampleAppInRepo())
	appId, err := common.AppRepo.GetAppId(tools.SampleMaintainer, tools.SampleApp)
	assert.Nil(t, err)

	backupClient := ProvideBackupClient()
	assert.Nil(t, backupClient.CreateBackup(appId, tools.AutoBackupDescription))
	assert.Nil(t, backupClient.CreateBackup(appId, tools.AutoBackupDescription))

	backups, err := backupClient.ListBackupsOfApp(tools.SampleAppBackupListRequestRemote)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(backups))

	assert.Nil(t, backupClient.RunRetentionPolicy())
	backups, err = backupClient.ListBackupsOfApp(tools.SampleAppBackupListRequestRemote)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(backups))
}
