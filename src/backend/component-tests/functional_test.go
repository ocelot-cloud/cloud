//go:build component

package component_tests

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/ocelot-cloud/shared/assert"
	"github.com/ocelot-cloud/shared/utils"
	"io"
	"net/http"
	"ocelot/backend/apps/backups"
	"ocelot/backend/apps/common"
	"ocelot/backend/certs"
	"ocelot/backend/clients"
	"ocelot/backend/ssh"
	"ocelot/backend/tools"
	"reflect"
	"strings"
	"testing"
	"time"
)

var (
	httpsHostUrl = getHttpsHostUrl()
	httpHost     = getHttpHost()
	httpsHost    = getHttpsHost()
)

func getHttpHost() string {
	if tools.Profile == tools.NATIVE {
		return "localhost:8080"
	} else {
		return "localhost"
	}
}
func getHttpsHost() string {
	if tools.Profile == tools.DOCKER_TEST {
		return "localhost"
	} else {
		return "localhost:8443"
	}
}

func getHttpsHostUrl() string {
	if tools.Profile == tools.DOCKER_TEST {
		return "https://localhost"
	} else {
		return "https://localhost:8443"
	}
}

func TestAssertContentUsingHttps(t *testing.T) {
	client := getClientAndLogin(t)
	defer client.wipeData()
	_, err := client.installSampleApp("2.0")
	assert.Nil(t, err)
	app := client.getInstalledSampleApp()
	client.setHostValue(httpHost)
	assert.Nil(t, client.startApp(app.AppId))
	if tools.Profile != tools.NATIVE {
		time.Sleep(1 * time.Second)
	}
	assert.Nil(t, client.assertContent("this is version 2.0"))

	client.setHostValue(httpsHost)
	client.parent.RootUrl = httpsHostUrl
	assert.Nil(t, client.assertContent("this is version 2.0"))
}

func TestCertificates(t *testing.T) {
	cleanupClient := getClient(t)
	defer cleanupClient.wipeData()

	var host string
	if tools.Profile == tools.DOCKER_TEST {
		host = "localhost:443"
	} else {
		host = "localhost:8443"
	}
	httpClient := getClient(t)
	defer httpClient.wipeData()

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	r, err := client.Post("https://"+host+"/api/login", "application/json", strings.NewReader(`{"username":"admin","password":"password"}`))
	assert.Nil(t, err)
	var authCookie *http.Cookie
	for _, c := range r.Cookies() {
		if c.Name == tools.OcelotAuthCookieName {
			authCookie = c
			break
		}
	}
	initialServerCert := grabServerCert(t, host)
	clientCert, err := certs.GenerateUniversalSelfSignedCert()
	assert.Nil(t, err)
	assert.NotEqual(t, initialServerCert, clientCert.Certificate[0])

	fullchainBytes, err := certs.ConvertToFullchainPemBytes(clientCert)
	assert.Nil(t, err)
	certificateBlob, err := json.Marshal(certs.Blob{Data: fullchainBytes})
	assert.Nil(t, err)

	certificateUploadUrl := fmt.Sprintf("https://%s%s", host, tools.SettingsCertificateUploadPath)
	req, err := http.NewRequest("POST", certificateUploadUrl, bytes.NewReader(certificateBlob))
	assert.Nil(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(authCookie)
	resp, err := client.Do(req)
	assert.Nil(t, err)
	defer utils.Close(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, "certificate uploaded successfully", string(body))

	newServerCert := grabServerCert(t, host)
	assert.Equal(t, newServerCert, clientCert.Certificate[0])
	assert.NotEqual(t, newServerCert, initialServerCert)
}

func grabServerCert(t *testing.T, host string) []byte {
	conn, err := tls.Dial("tcp", host, &tls.Config{InsecureSkipVerify: true})
	assert.Nil(t, err)
	defer func(conn *tls.Conn) {
		assert.Nil(t, conn.Close())
	}(conn)

	serverCertificates := conn.ConnectionState().PeerCertificates
	assert.Equal(t, 1, len(serverCertificates))

	return serverCertificates[0].Raw
}

func TestAppStore(t *testing.T) {
	client := getClientAndLogin(t)
	defer client.wipeData()
	storeApp, err := client.searchForSampleApp()
	assert.Nil(t, err)
	assert.Equal(t, tools.SampleApp, storeApp.AppName)
	assert.Equal(t, tools.SampleMaintainer, storeApp.Maintainer)
	assert.Equal(t, "2.0", storeApp.LatestVersionName)

	latestAppVersion := client.findVersion(storeApp.AppId, "2.0")
	assert.Equal(t, "2.0", latestAppVersion.Name)
	assert.Equal(t, tools.SampleAppVersion2CreationTimestamp, latestAppVersion.VersionCreationTimestamp)
	assert.Equal(t, storeApp.LatestVersionId, latestAppVersion.Id)

	oldAppVersion := client.findVersion(storeApp.AppId, "1.0")
	assert.Equal(t, "1.0", oldAppVersion.Name)
	assert.Equal(t, tools.SampleAppVersion1CreationTimestamp, oldAppVersion.VersionCreationTimestamp)
	assert.NotEqual(t, storeApp.LatestVersionId, oldAppVersion.Id)
}

func TestManualBackup(t *testing.T) {
	client := getClientAndLogin(t)
	defer client.wipeData()
	_, err := client.installSampleApp("2.0")
	assert.Nil(t, err)

	appBackups := client.listAppBackups(tools.SampleMaintainer, tools.SampleApp, true)
	assert.Equal(t, 0, len(appBackups))
	client.createSampleAppBackup()
	appBackups = client.listAppBackups(tools.SampleMaintainer, tools.SampleApp, true)
	assert.Equal(t, 1, len(appBackups))

	backup := appBackups[0]
	assert.Equal(t, tools.SampleMaintainer, backup.Maintainer)
	assert.Equal(t, tools.SampleApp, backup.AppName)
	assert.Equal(t, "2.0", backup.VersionName)
	assert.Equal(t, tools.ManualBackupDescription, backup.Description)
	assert.Equal(t, tools.SampleAppVersion2CreationTimestamp, backup.VersionCreationTimestamp)
	assert.True(t, backup.BackupCreationTimestamp.Before(time.Now()))
	assert.True(t, backup.BackupCreationTimestamp.After(time.Now().Add(-1*time.Minute)))
}

func TestStartingAndStoppingApps(t *testing.T) {
	client := getClientAndLogin(t)
	defer client.wipeData()
	_, err := client.installSampleApp("2.0")
	assert.Nil(t, err)
	sampleApp := client.getInstalledSampleApp()

	assert.Equal(t, "Uninitialized", sampleApp.Status)
	assert.Nil(t, client.startApp(sampleApp.AppId))
	time.Sleep(1 * time.Second)
	sampleApp = client.getInstalledSampleApp()
	assert.Equal(t, "Available", sampleApp.Status)
}

func TestUpdatesAndPreUpdateBackupCreation(t *testing.T) {
	client := getClientAndLogin(t)
	defer client.wipeData()
	installedSampleApp, err := client.installSampleApp("1.0")
	assert.Nil(t, err)

	client.setHostValue("localhost")

	assert.Nil(t, client.startApp(installedSampleApp.AppId))
	time.Sleep(1 * time.Second)
	assert.Nil(t, client.assertContent("this is version 1.0"))

	appBackups := client.listAppBackups(tools.SampleMaintainer, tools.SampleApp, true)
	assert.Equal(t, 0, len(appBackups))

	assert.Nil(t, client.updateApp(installedSampleApp.AppId))
	installedSampleApp = client.getInstalledSampleApp()
	assert.Equal(t, "2.0", installedSampleApp.VersionName)
	assert.Nil(t, client.assertContent("this is version 2.0"))

	appBackups = client.listAppBackups(tools.SampleMaintainer, tools.SampleApp, true)
	assert.Equal(t, 1, len(appBackups))
	backup := appBackups[0]
	assert.Equal(t, "1.0", backup.VersionName)
	assert.Equal(t, tools.AutoBackupDescription, backup.Description)
	assert.Equal(t, tools.SampleAppVersion1CreationTimestamp, backup.VersionCreationTimestamp)
	assert.Equal(t, tools.SampleApp, backup.AppName)
	assert.Equal(t, tools.SampleMaintainer, backup.Maintainer)

	client.restoreBackup(backup.BackupId, true)
	installedSampleApp = client.getInstalledSampleApp()
	assert.Equal(t, "1.0", installedSampleApp.VersionName)
	assert.Nil(t, client.assertContent("this is version 1.0"))
}

func TestUsers(t *testing.T) {
	client := getClientAndLogin(t)
	defer client.wipeData()

	users := client.getUsers()
	assert.Equal(t, 1, len(users))
	adminUser := users[0]
	assert.Equal(t, "admin", adminUser.Name)
	assert.Equal(t, "admin", adminUser.Role)

	assert.Nil(t, client.createUser("user", "userpassword"))
	users = client.getUsers()
	assert.Equal(t, 2, len(users))
	normalUser, err := client.getNormalUser(users)
	assert.Nil(t, err)
	assert.Equal(t, "user", normalUser.Name)
	assert.Equal(t, "user", normalUser.Role)

	assert.Nil(t, client.deleteUser(normalUser.Id))
	users = client.getUsers()
	assert.Equal(t, 1, len(users))
	adminUser = users[0]
	assert.Equal(t, "admin", adminUser.Name)
	assert.Equal(t, "admin", adminUser.Role)
}

func TestPostgresAppOperations(t *testing.T) {
	client := getClientAndLogin(t)
	defer client.wipeData()
	assert.Nil(t, client.createUser("user", "userpassword"))
	apps := client.listInstalledApps()
	var postgresApp tools.AppDto
	for _, app := range apps {
		if app.AppName == tools.OcelotDbAppName {
			postgresApp = app
		}
	}
	assert.Equal(t, "Available", postgresApp.Status)
	assert.Equal(t, tools.OcelotDbMaintainer, postgresApp.Maintainer)
	assert.Equal(t, tools.OcelotDbAppName, postgresApp.AppName)
	assert.Equal(t, common.PostgresVersion, postgresApp.VersionName)

	backupInfos := client.listAppBackups(postgresApp.Maintainer, postgresApp.AppName, true)
	for _, backupInfo := range backupInfos {
		client.deleteBackup(backupInfo.BackupId, true)
	}

	backupInfos = client.listAppBackups(postgresApp.Maintainer, postgresApp.AppName, true)
	assert.Equal(t, 0, len(backupInfos))
	client.createBackup(postgresApp.AppId)
	backupInfos = client.listAppBackups(postgresApp.Maintainer, postgresApp.AppName, true)
	assert.Equal(t, 1, len(backupInfos))
	backup := backupInfos[0]
	defer client.deleteBackup(backup.BackupId, true)

	assert.Equal(t, 2, len(client.getUsers()))
	client.deleteUserIfPresent()
	assert.Equal(t, 1, len(client.getUsers()))

	client.restoreBackup(backup.BackupId, true)
	assert.Equal(t, 2, len(client.getUsers()))
}

func TestUserAccess(t *testing.T) {
	adminClient := getClientAndLogin(t)
	defer adminClient.wipeData()
	adminClient.setHostValue("localhost")
	sampleApp, err := adminClient.installSampleApp("2.0")
	assert.Nil(t, err)

	assert.Nil(t, adminClient.createUser("user", "userpassword"))
	userClient := getClient(t)
	userClient.parent.User = "user"
	userClient.parent.Password = "userpassword"
	assert.Nil(t, userClient.login())

	apps := adminClient.listInstalledApps()
	assert.Equal(t, 2, len(apps))

	installedApps := userClient.listInstalledApps()
	assert.Equal(t, 0, len(installedApps))
	assert.Nil(t, adminClient.startApp(sampleApp.AppId))
	installedApps = userClient.listInstalledApps()
	assert.Equal(t, 1, len(installedApps))
	assert.Nil(t, adminClient.assertContent("this is version 2.0"))
}

func TestLogout(t *testing.T) {
	client := getClientAndLogin(t)
	assert.Nil(t, client.logout())
	err := client.logout()
	assert.NotNil(t, err)
	assert.Equal(t, utils.GetErrMsg(400, "cookie not found"), err.Error())
}

func TestHostSettings(t *testing.T) {
	client := getClientAndLogin(t)
	defer client.wipeData()
	client.setHostValue("localhost")
	host, err := client.getHostValue()
	assert.Nil(t, err)
	assert.Equal(t, "localhost", host)
	client.setHostValue("localhost2")
	host, err = client.getHostValue()
	assert.Nil(t, err)
	assert.Equal(t, "localhost2", host)
}

func TestSshConfigsSetting(t *testing.T) {
	client := getClientAndLogin(t)
	defer client.wipeData()
	initialConfigs, err := client.readSshConfigs()
	assert.Nil(t, err)
	reflect.DeepEqual(initialConfigs, ssh.GetSampleRemoteRepo())

	client.saveSshConfigs(ssh.GetSampleRemoteRepo())
	configs, err := client.readSshConfigs()
	assert.Nil(t, err)
	reflect.DeepEqual(configs, ssh.GetSampleRemoteRepo())
}

func TestRemoteBackups(t *testing.T) {
	// the ssh container is available on localhost:2222 which is not available in the dockerized backend, so we skip that test
	if tools.Profile != tools.NATIVE {
		t.Skip()
		return
	}
	ssh.ShutDownSshTestContainer()
	assert.Nil(t, utils.ExecuteShellCommand("docker volume rm backups || true"))
	defer ssh.ShutDownSshTestContainer()
	ssh.StartSshTestContainer()

	client := getClientAndLogin(t)
	defer client.wipeData()
	_, err := client.installSampleApp("2.0")
	assert.Nil(t, err)
	assertBackupNumbers(client, 0, 0)
	appId := client.getInstalledSampleApp().AppId
	client.createBackup(appId)
	assertBackupNumbers(client, 1, 0)

	enableRemoteBackupRepo(client)
	client.createBackup(appId)
	assertBackupNumbers(client, 2, 1)

	backupInfos := client.listAppBackups(tools.SampleMaintainer, tools.SampleApp, false)
	assert.Equal(t, 1, len(backupInfos))
	backup := backupInfos[0]

	client.restoreBackup(backup.BackupId, false)
	client.deleteBackup(backup.BackupId, false)
	assertBackupNumbers(client, 2, 0)
}

func enableRemoteBackupRepo(client *CloudClient) {
	repo := ssh.GetSampleRemoteRepo()
	assert.NotNil(client.t, client.testSshAccess(repo))
	knownHosts := client.getKnownHosts(repo)
	assert.True(client.t, strings.Contains(knownHosts, "[localhost]:2222"))
	repo.SshKnownHosts = knownHosts
	assert.Nil(client.t, client.testSshAccess(repo))
	client.saveSshConfigs(repo)
}

func assertBackupNumbers(client *CloudClient, expectedLocalBackups, expectedRemoteBackups int) {
	tools.Logger.Info("Asserting local backup number")
	appBackups := client.listAppBackups(tools.SampleMaintainer, tools.SampleApp, true)
	assert.Equal(client.t, expectedLocalBackups, len(appBackups))
	tools.Logger.Info("Asserting remote backup number")
	appBackups = client.listAppBackups(tools.SampleMaintainer, tools.SampleApp, false)
	assert.Equal(client.t, expectedRemoteBackups, len(appBackups))
}

func TestCheckAuth(t *testing.T) {
	client := getClientAndLogin(t)
	defer client.wipeData()
	authorization := client.checkAuth()
	assert.Equal(t, "admin", authorization.User)
	assert.True(t, authorization.IsAdmin)

	assert.Nil(t, client.createUser("user", "userpassword"))
	client.parent.User = "user"
	client.parent.Password = "userpassword"
	assert.Nil(t, client.login())
	authorization = client.checkAuth()
	assert.Equal(t, "user", authorization.User)
	assert.False(t, authorization.IsAdmin)
}

func TestLogin(t *testing.T) {
	client := getClient(t)
	defer client.wipeData()
	assert.Nil(t, client.parent.Cookie)
	assert.Nil(t, client.login())
	cookie := client.parent.Cookie
	assert.NotNil(t, cookie)
	assert.Equal(t, 64, len(cookie.Value))
	assert.True(t, cookie.Expires.After(time.Now().AddDate(0, 0, 29)))
	assert.True(t, cookie.Expires.Before(time.Now().AddDate(0, 0, 31)))
}

func TestCookieExpirationDateRenewal(t *testing.T) {
	if tools.Profile != tools.DOCKER_TEST {
		t.Skip()
		return
	}
	client := getClientAndLogin(t)
	defer client.wipeData()

	cookieExpiration1 := client.checkAuth().CookieExpirationDate
	assert.True(t, cookieExpiration1.After(time.Now().AddDate(0, 0, 29)))
	assert.True(t, cookieExpiration1.Before(time.Now().AddDate(0, 0, 31)))

	time.Sleep(1 * time.Second)
	cookieExpiration2 := client.checkAuth().CookieExpirationDate
	assert.True(t, cookieExpiration1.Before(cookieExpiration2))
}

func TestStopStackNotExisting(t *testing.T) {
	cloud := getClientAndLogin(t)
	notExistingId := "123"
	err := cloud.stopApp(notExistingId)
	assert.NotNil(t, err)
	assert.Equal(t, utils.GetErrMsg(400, "Failed to get app"), err.Error())
}

func TestProhibitedOperationsOnPostgresApp(t *testing.T) {
	cloud := getClientAndLogin(t)
	defer cloud.wipeData()
	apps := cloud.listInstalledApps()
	postgresApp := apps[0]

	expectedErrorMessage := "operation not allowed on postgres app"
	err := cloud.startApp(postgresApp.AppId)
	assert.NotNil(t, err)
	assert.Equal(t, utils.GetErrMsg(401, expectedErrorMessage), err.Error())

	err = cloud.updateApp(postgresApp.AppId)
	assert.NotNil(t, err)
	assert.Equal(t, utils.GetErrMsg(401, expectedErrorMessage), err.Error())

	err = cloud.pruneApp(postgresApp.AppId)
	assert.NotNil(t, err)
	assert.Equal(t, utils.GetErrMsg(401, expectedErrorMessage), err.Error())
}

func TestInstallingExistingAppShouldFail(t *testing.T) {
	cloud := getClientAndLogin(t)
	defer cloud.wipeData()
	_, err := cloud.installSampleApp("2.0")
	assert.Nil(t, err)
	_, err = cloud.installSampleApp("2.0")
	assert.NotNil(t, err)
	assert.Equal(t, utils.GetErrMsg(409, "can't install app 'samplemaintainer / sampleapp' because it is already installed"), err.Error())
}

func TestErrorWhenUpdatingAndLatestVersionAlreadyInstalled(t *testing.T) {
	cloud := getClientAndLogin(t)
	defer cloud.wipeData()
	app, err := cloud.installSampleApp("2.0")
	assert.Nil(t, err)
	err = cloud.updateApp(app.AppId)
	assert.NotNil(t, err)

	expectedAppInfo := tools.RepoApp{
		Maintainer:  tools.SampleMaintainer,
		AppName:     tools.SampleApp,
		VersionName: "2.0",
	}
	expectedErrorMessage := clients.GetUpdateErrorString(expectedAppInfo)
	assert.True(t, strings.Contains(err.Error(), expectedErrorMessage))
}

func TestAppStoreSearch(t *testing.T) {
	cloud := getClientAndLogin(t)
	defer cloud.wipeData()
	app, err := cloud.searchForSampleApp()
	assert.Nil(t, err)
	assert.NotNil(t, app)

	cloud.searchForUnofficialApps = false
	app, err = cloud.searchForSampleApp()
	assert.NotNil(t, err)
	assert.Equal(t, utils.GetErrMsg(404, "no apps found"), err.Error())
	assert.Nil(t, app)
}

func TestAppListingInBackupRepo(t *testing.T) {
	if tools.Profile != tools.NATIVE {
		t.Skip()
		return
	}
	ssh.ShutDownSshTestContainer()
	assert.Nil(t, utils.ExecuteShellCommand("docker volume rm backups || true"))
	defer ssh.ShutDownSshTestContainer()
	ssh.StartSshTestContainer()

	cloud := getClientAndLogin(t)
	defer cloud.wipeData()
	assertAppsNumbersInBackupRepo(cloud, 0, 0)

	_, err := cloud.installSampleApp("2.0")
	assert.Nil(t, err)
	cloud.createSampleAppBackup()

	assertAppsNumbersInBackupRepo(cloud, 1, 0)

	enableRemoteBackupRepo(cloud)
	cloud.createSampleAppBackup()

	assertAppsNumbersInBackupRepo(cloud, 1, 1)
}

func assertAppsNumbersInBackupRepo(cloud *CloudClient, expectedAppNumberInLocalBackupRepo, expectedAppNumberInRemoteBackupRepo int) {
	apps, err := cloud.listAppsInBackupRepo(true)
	assert.Nil(cloud.t, err)
	assert.Equal(cloud.t, expectedAppNumberInLocalBackupRepo, len(apps))
	apps, err = cloud.listAppsInBackupRepo(false)
	assert.Nil(cloud.t, err)
	assert.Equal(cloud.t, expectedAppNumberInRemoteBackupRepo, len(apps))
}

func TestAdminCantDeleteHisOwnAccount(t *testing.T) {
	cloud := getClientAndLogin(t)
	defer cloud.wipeData()
	users := cloud.getUsers()
	assert.Equal(t, 1, len(users))
	adminUserId := users[0].Id
	err := cloud.deleteUser(adminUserId)
	assert.NotNil(t, err)
	assert.Equal(t, utils.GetErrMsg(401, "an admin can not delete his own account"), err.Error())
}

func TestMaintenanceSettings(t *testing.T) {
	cloud := getClientAndLogin(t)
	defer cloud.wipeData()

	maintenanceSettingsFromServer := cloud.getMaintenanceSettings()
	assert.Equal(t, 4, maintenanceSettingsFromServer.PreferredMaintenanceHour)
	assert.Equal(t, true, maintenanceSettingsFromServer.AreAutoBackupsEnabled)
	assert.Equal(t, true, maintenanceSettingsFromServer.AreAutoUpdatesEnabled)

	newMaintenanceSettings := backups.MaintenanceSettings{
		AreAutoBackupsEnabled:    false,
		AreAutoUpdatesEnabled:    false,
		PreferredMaintenanceHour: 7,
	}
	cloud.setMaintenanceSettings(newMaintenanceSettings)

	maintenanceSettingsFromServer = cloud.getMaintenanceSettings()
	assert.Equal(t, 7, maintenanceSettingsFromServer.PreferredMaintenanceHour)
	assert.Equal(t, false, maintenanceSettingsFromServer.AreAutoBackupsEnabled)
	assert.Equal(t, false, maintenanceSettingsFromServer.AreAutoUpdatesEnabled)
}

func TestNullOriginHeaderIsAllowed(t *testing.T) {
	cloud := getClientAndLogin(t)
	defer cloud.wipeData()
	cloud.parent.Origin = "null"
	assert.Nil(t, cloud.createUser("user", "password"))
}

func TestDeletingApp(t *testing.T) {
	cloud := getClientAndLogin(t)
	defer cloud.wipeData()

	apps := cloud.listInstalledApps()
	assert.Equal(t, 1, len(apps))

	_, err := cloud.installSampleApp("2.0")
	assert.Nil(t, err)

	apps = cloud.listInstalledApps()
	assert.Equal(t, 2, len(apps))

	app := cloud.getInstalledSampleApp()

	assert.Nil(t, cloud.pruneApp(app.AppId))
	apps = cloud.listInstalledApps()
	assert.Equal(t, 1, len(apps))
}

func TestChangePassword(t *testing.T) {
	adminClient := getClientAndLogin(t)
	defer adminClient.wipeData()

	assert.Nil(t, adminClient.createUser("user", "userpassword"))
	userClient := getClient(t)
	userClient.parent.User = "user"
	userClient.parent.Password = "userpassword"
	assert.Nil(t, userClient.login())

	oldPassword := userClient.parent.Password
	newPassword := oldPassword + "2"
	assert.Nil(t, userClient.changePassword(newPassword))
	assert.NotNil(t, userClient.login())
	userClient.parent.Password = newPassword
	assert.Nil(t, userClient.login())

	err := userClient.changePassword(oldPassword + " ")
	assert.NotNil(t, err)
	assert.Equal(t, utils.GetErrMsg(400, "invalid input"), err.Error())
}

func TestAppDownload(t *testing.T) {
	cloud := getClientAndLogin(t)
	defer cloud.wipeData()
	appDownload, err := cloud.downloadApp()
	assert.Nil(t, err)
	assert.Equal(t, "2.0.zip", appDownload.FileName)
	assert.Equal(t, len(tools.GetSampleAppContent()), len(appDownload.Content))
}
