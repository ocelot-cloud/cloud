//go:build slow

package backups

import (
	"fmt"
	"github.com/ocelot-cloud/shared/assert"
	tr "github.com/ocelot-cloud/task-runner"
	"ocelot/backend/apps/cloud"
	"ocelot/backend/apps/common"
	"ocelot/backend/clients"
	"ocelot/backend/security"
	"ocelot/backend/ssh"
	"ocelot/backend/tools"
	"strings"
	"testing"
	"time"
)

var (
	createBackupTestFiles     = `mkdir -p /home/myuser/testfolder && echo hello > /home/myuser/testfolder/hello.txt`
	showBackupTestFile        = "cat testfolder/hello.txt"
	listBackupTestFile        = "ls -lh testfolder | tail -n +2"
	listBackupTestFileWithIds = "ls -ln testfolder | tail -n +2"
)

func TestMain(m *testing.M) {
	tools.Config.IsUsingDockerNetwork = false
	tools.Config.UseProductionDatabaseContainer = false
	common.InitializeDatabase(false, false)
	clients.Apps = cloud.GetAppManager()
	clients.BackupManager = ProvideBackupClient()
	defer common.WipeWholeDatabase()
	m.Run()
}

func setup() {
	cleanup()
	cloud.CreateExternalDockerNetworkAndConnectOcelotCloud(tools.OcelotDbMaintainer, tools.ResticAppName)
	err := PrepareLocalBackupContainer()
	if err != nil {
		Logger.Fatal("Failed to prepare backup container and repos: %v", err)
	}
	tr.ExecuteInDir(tools.DockerDir, createCreateSampleappImageCommand)
}

func executeInAppContainer(command string) (string, error) {
	wholeCommand := fmt.Sprintf(`docker run --rm -v %s:/home/myuser --entrypoint "" %s sh -c "%s"`, tools.SampleAppDockerVolume, fileBackupTestAppImageName, command)
	output, err := runCommandWithOutputString(wholeCommand)
	if err != nil {
		return output, err
	}
	return output, err
}

func TestBackupApiLocally(t *testing.T) {
	setup()
	defer cleanup()
	assert.Nil(t, common.CreateSampleAppInRepo())

	_, err := executeInAppContainer(createBackupTestFiles)
	assert.Nil(t, err)
	assertTestFileCorrectness(t)

	assertNoBackupsPresent(t, tools.SampleAppBackupListRequestLocal)

	backupVersionInfo := createAndAssertBackup(t, tools.SampleAppBackupListRequestLocal)
	restoreBackupAndAssertRestoredFiles(t, backupVersionInfo.BackupId)

	assert.Nil(t, clients.BackupManager.DeleteBackup(backupVersionInfo.BackupId, true))
	assertNoBackupsPresent(t, tools.SampleAppBackupListRequestLocal)
}

func cleanup() {
	common.WipeWholeDatabase()
	err := runCommand("docker rm -f " + tools.SampleAppDockerContainer)
	if err != nil {
		Logger.Info("no sample app container to remove")
	}
	command := fmt.Sprintf("docker volume rm %s backups restic_ssh restic_rclone", tools.SampleAppDockerVolume)
	err = runCommand(command)
	if err != nil {
		Logger.Info("no backup volume to remove")
	}
}

func assertNoBackupsPresent(t *testing.T, sampleBackupRequest tools.BackupListRequest) {
	backups, err := clients.BackupManager.ListBackupsOfApp(sampleBackupRequest)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(backups))
}

func restoreBackupAndAssertRestoredFiles(t *testing.T, backupId string) {
	/* TODO?
	_, err := executeInAppContainer(`[ -d "testfolder" ] || exit 1`)
	assert.Nil(t, err)
	*/
	assert.Nil(t, runCommand("docker rm -f "+tools.SampleAppDockerContainer))
	assert.Nil(t, runCommand("docker volume rm "+tools.SampleAppDockerVolume))
	_, err := executeInAppContainer(`[ ! -d "testfolder" ] || exit 1`)
	assert.Nil(t, err)
	backupInfo, err := clients.BackupManager.RestoreBackup(tools.BackupOperationRequest{
		BackupId: backupId,
		IsLocal:  true,
	})
	assert.Nil(t, err)
	assertTestFileCorrectness(t)
	assert.Equal(t, tools.SampleMaintainer, backupInfo.Maintainer)
	assert.Equal(t, tools.SampleApp, backupInfo.AppName)
	assert.Equal(t, tools.SampleAppVersion1Name, backupInfo.VersionName)
	assert.Equal(t, tools.GetSampleAppContent(), backupInfo.VersionContent)
}

func assertTestFileCorrectness(t *testing.T) {
	testFileContent, err := executeInAppContainer(showBackupTestFile)
	assert.Nil(t, err)
	assert.Equal(t, testFileContent, "hello\n")

	fileList, err := executeInAppContainer(listBackupTestFile)
	assert.Nil(t, err)
	assert.True(t, strings.HasPrefix(fileList, "-rw-r--r--    1 myuser   mygroup"))
	assert.True(t, strings.HasSuffix(fileList, "hello.txt\n"))

	fileList, err = executeInAppContainer(listBackupTestFileWithIds)
	assert.Nil(t, err)
	assert.True(t, strings.HasPrefix(fileList, "-rw-r--r--    1 1001     1001"))
	assert.True(t, strings.HasSuffix(fileList, "hello.txt\n"))
}

func createAndAssertBackup(t *testing.T, sampleBackupRequest tools.BackupListRequest) tools.BackupInfo {
	appId, err := common.AppRepo.GetAppId(sampleBackupRequest.Maintainer, sampleBackupRequest.AppName)
	assert.Nil(t, err)
	assert.Nil(t, clients.BackupManager.CreateBackup(appId, tools.SampleBackupDescription))

	backups, err := clients.BackupManager.ListBackupsOfApp(sampleBackupRequest)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(backups))

	backupInfo := backups[0]
	assert.Equal(t, tools.SampleMaintainer, backupInfo.Maintainer)
	assert.Equal(t, tools.SampleApp, backupInfo.AppName)
	assert.Equal(t, tools.SampleAppVersion1Name, backupInfo.VersionName)
	assert.Equal(t, tools.SampleAppVersion1CreationTimestamp, backupInfo.VersionCreationTimestamp)
	assert.Equal(t, tools.SampleBackupDescription, backupInfo.Description)
	assert.True(t, backupInfo.BackupCreationTimestamp.After(time.Now().UTC().Add(-30*time.Second)))
	assert.True(t, backupInfo.BackupCreationTimestamp.Before(time.Now().UTC().Add(30*time.Second)))
	return backupInfo
}

func TestBackupOfOcelotCloudDatabase(t *testing.T) {
	cleanup()
	assert.Nil(t, PrepareLocalBackupContainer())
	defer func() {
		_ = runCommand("docker volume rm backups || true")
	}()

	assert.Nil(t, security.UserRepo.CreateUser("testuser", "testpassword", false))

	appId, err := common.AppRepo.GetAppId(tools.OcelotDbMaintainer, tools.OcelotDbAppName)
	assert.Nil(t, err)
	assert.Nil(t, clients.BackupManager.CreateBackup(appId, "testing-ocelotcloud-database-backup"))
	appBackup, err := clients.BackupManager.ListBackupsOfApp(tools.OcelotDbAppBackupListRequestLocal)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(appBackup))
	assert.Equal(t, tools.OcelotDbMaintainer, appBackup[0].Maintainer)
	assert.Equal(t, tools.OcelotDbAppName, appBackup[0].AppName)

	userId, err := security.UserRepo.GetUserId("testuser")
	assert.Nil(t, err)
	assert.Nil(t, security.UserRepo.DeleteUser(userId))
	assert.False(t, security.UserRepo.DoesUserExist("testuser"))

	_, err = clients.BackupManager.RestoreBackup(tools.BackupOperationRequest{
		BackupId: appBackup[0].BackupId,
		IsLocal:  true,
	})
	assert.Nil(t, err)
	assert.True(t, security.UserRepo.DoesUserExist("testuser"))
}

func TestRemoteRepoBackupAcceptance(t *testing.T) {
	setup()
	defer cleanup()
	defer ssh.ShutDownSshTestContainer()
	ssh.StartSshTestContainer()
	common.InitializeDatabase(false, false)
	defer common.WipeWholeDatabase()

	err := security.UserRepo.CreateUser(tools.SampleMaintainer, tools.SampleApp, false)
	assert.Nil(t, err)
	assert.True(t, security.UserRepo.DoesUserExist(tools.SampleMaintainer))

	remoteRepo := ssh.GetSampleRemoteRepo()
	assert.Nil(t, ssh.SetRemoteBackupRepository(remoteRepo))

	postgresAppId, err := common.AppRepo.GetAppId(tools.OcelotDbMaintainer, tools.OcelotDbAppName)
	assert.Nil(t, err)
	_, err = clients.BackupManager.ListBackupsOfApp(tools.OcelotDbAppBackupListRequestRemote)
	assert.NotNil(t, err)
	err = clients.BackupManager.CreateBackup(postgresAppId, "testing-ocelotcloud-database-backup")
	assert.NotNil(t, err)

	knownHosts, err := ssh.SshClient.GetKnownHosts(remoteRepo.Host, remoteRepo.SshPort)
	assert.Nil(t, err)
	remoteRepo.SshKnownHosts = knownHosts
	assert.Nil(t, ssh.SetRemoteBackupRepository(remoteRepo))

	backups, err := clients.BackupManager.ListBackupsOfApp(tools.OcelotDbAppBackupListRequestRemote)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(backups))
	err = clients.BackupManager.CreateBackup(postgresAppId, "testing-ocelotcloud-database-backup")
	assert.Nil(t, err)

	assert.True(t, security.UserRepo.DoesUserExist(tools.SampleMaintainer))
	userId, err := security.UserRepo.GetUserId(tools.SampleMaintainer)
	assert.Nil(t, err)
	assert.Nil(t, security.UserRepo.DeleteUser(userId))
	assert.False(t, security.UserRepo.DoesUserExist(tools.SampleMaintainer))

	backups, err = clients.BackupManager.ListBackupsOfApp(tools.OcelotDbAppBackupListRequestRemote)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(backups))
	backup := backups[0]

	remoteRepo.SshKnownHosts = "sample-string"
	assert.Nil(t, ssh.SetRemoteBackupRepository(remoteRepo))
	_, err = clients.BackupManager.RestoreBackup(tools.BackupOperationRequest{
		BackupId: backup.BackupId,
		IsLocal:  false,
	})
	assert.NotNil(t, err)

	remoteRepo.SshKnownHosts = knownHosts
	assert.Nil(t, ssh.SetRemoteBackupRepository(remoteRepo))
	_, err = clients.BackupManager.RestoreBackup(tools.BackupOperationRequest{
		BackupId: backup.BackupId,
		IsLocal:  false,
	})
	assert.Nil(t, err)
	assert.True(t, security.UserRepo.DoesUserExist(tools.SampleMaintainer))

	remoteRepo.SshKnownHosts = "sample-string"
	assert.Nil(t, ssh.SetRemoteBackupRepository(remoteRepo))
	err = clients.BackupManager.DeleteBackup(backup.BackupId, false)
	assert.NotNil(t, err)

	remoteRepo.SshKnownHosts = knownHosts
	assert.Nil(t, ssh.SetRemoteBackupRepository(remoteRepo))
	err = clients.BackupManager.DeleteBackup(backup.BackupId, false)
	assert.Nil(t, err)
	backups, err = clients.BackupManager.ListBackupsOfApp(tools.OcelotDbAppBackupListRequestRemote)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(backups))
}

func TestShowCommandOutput(t *testing.T) {
	assert.False(t, showCommandOutput)
}

func TestFindUniqueMaintainerAndAppNamePairs(t *testing.T) {
	input := []tools.MaintainerAndApp{
		{Maintainer: "a", AppName: "x"},
		{Maintainer: "a", AppName: "y"},
		{Maintainer: "b", AppName: "y"},
		{Maintainer: "a", AppName: "x"},
		{Maintainer: "a", AppName: "z"},
	}

	expected := []tools.MaintainerAndApp{
		{Maintainer: "a", AppName: "x"},
		{Maintainer: "a", AppName: "y"},
		{Maintainer: "b", AppName: "y"},
		{Maintainer: "a", AppName: "z"},
	}

	actual := tools.FindUniqueMaintainerAndAppNamePairs(input)
	assert.Equal(t, expected, actual)
}

func TestListAppsInBackupRepo(t *testing.T) {
	setup()
	defer cleanup()
	defer ssh.ShutDownSshTestContainer()
	ssh.StartSshTestContainer()
	common.InitializeDatabase(false, false)
	defer common.WipeWholeDatabase()

	assert.Nil(t, common.CreateSampleAppInRepo())
	sampleAppId, err := common.AppRepo.GetAppId(tools.SampleMaintainer, tools.SampleApp)
	assert.Nil(t, err)
	assert.Nil(t, clients.Apps.StartApp(sampleAppId))
	ocelotAppId, err := common.AppRepo.GetAppId(tools.OcelotDbMaintainer, tools.OcelotDbAppName)
	assert.Nil(t, err)

	repo, err := clients.BackupManager.ListAppsInBackupRepo(true)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(repo))
	repo, err = clients.BackupManager.ListAppsInBackupRepo(false)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(repo))

	err = clients.BackupManager.CreateBackup(ocelotAppId, "testing-ocelotcloud-database-backup")
	assert.Nil(t, err)
	repo, err = clients.BackupManager.ListAppsInBackupRepo(true)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(repo))
	assert.Equal(t, tools.OcelotDbMaintainer, repo[0].Maintainer)
	assert.Equal(t, tools.OcelotDbAppName, repo[0].AppName)
	repo, err = clients.BackupManager.ListAppsInBackupRepo(false)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(repo))

	remoteRepo := ssh.GetSampleRemoteRepo()
	assert.Nil(t, ssh.SetRemoteBackupRepository(remoteRepo))
	knownHosts, err := ssh.SshClient.GetKnownHosts(remoteRepo.Host, remoteRepo.SshPort)
	assert.Nil(t, err)
	remoteRepo.SshKnownHosts = knownHosts
	assert.Nil(t, ssh.SetRemoteBackupRepository(remoteRepo))

	err = clients.BackupManager.CreateBackup(sampleAppId, "testing-ocelotcloud-database-backup")
	assert.Nil(t, err)
	repo, err = clients.BackupManager.ListAppsInBackupRepo(true)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(repo))
	assert.Equal(t, tools.OcelotDbMaintainer, repo[0].Maintainer)
	assert.Equal(t, tools.OcelotDbAppName, repo[0].AppName)
	assert.Equal(t, tools.SampleMaintainer, repo[1].Maintainer)
	assert.Equal(t, tools.SampleApp, repo[1].AppName)
	repo, err = clients.BackupManager.ListAppsInBackupRepo(false)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(repo))
	assert.Equal(t, tools.SampleMaintainer, repo[0].Maintainer)
	assert.Equal(t, tools.SampleApp, repo[0].AppName)
}
