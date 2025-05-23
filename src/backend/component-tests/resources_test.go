package component_tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ocelot-cloud/shared/assert"
	"github.com/ocelot-cloud/shared/utils"
	"io"
	"net/http"
	"ocelot/backend/apps/backups"
	"ocelot/backend/apps/store"
	"ocelot/backend/security"
	"ocelot/backend/tools"
	"os"
	"testing"
)

var logger = tools.Logger

type CloudClient struct {
	parent                  utils.ComponentClient
	appToOperateOn          string
	t                       *testing.T
	searchForUnofficialApps bool
}

func (c *CloudClient) getSecret() (string, error) {
	body, err := c.parent.DoRequest(tools.SecretPath, nil, "")
	if err != nil {
		return "", err
	}

	var secret string
	err = json.Unmarshal(body, &secret)
	if err != nil {
		return "", err
	}

	return secret, nil
}

func (c *CloudClient) logout() error {
	_, err := c.parent.DoRequest(tools.UsersLogoutPath, nil, "")
	if err != nil {
		return err
	}
	return nil
}

func (c *CloudClient) login() error {
	loginCredentials := security.Credentials{Username: c.parent.User, Password: c.parent.Password}
	_, err := c.parent.DoRequest(tools.LoginPath, loginCredentials, "")
	return err
}

func (c *CloudClient) wipeData() {
	_, err := c.parent.DoRequest(tools.WipePath, nil, "")
	assert.Nil(c.t, err)
}

func getClientAndLogin(t *testing.T) *CloudClient {
	cloud := getClient(t)
	assert.Nil(t, cloud.login())
	return cloud
}

func getClient(t *testing.T) *CloudClient {
	cloud := &CloudClient{
		parent: utils.ComponentClient{
			User:              "admin",
			Password:          "password",
			NewPassword:       "password" + "x",
			Cookie:            nil,
			SetCookieHeader:   true,
			RootUrl:           "http://localhost:8080",
			VerifyCertificate: false,
		},
		appToOperateOn:          tools.SampleApp,
		t:                       t,
		searchForUnofficialApps: true,
	}

	// When starting a test in the ID, the PROFILE env is empty. So the second expression is a convenience for running tests against the native running backend.
	if tools.Profile != tools.NATIVE && os.Getenv("PROFILE") != "" {
		cloud.parent.RootUrl = "http://ocelot-cloud.localhost"
	}

	return cloud
}

func (c *CloudClient) installSampleApp(version string) (*tools.AppDto, error) {
	storeApp, err := c.searchForSampleApp()
	if err != nil {
		return nil, err
	}
	versionInfo := c.findVersion(storeApp.AppId, version)
	_, err = c.parent.DoRequest(tools.VersionsInstallPath, tools.NumberString{Value: versionInfo.Id}, "")
	if err != nil {
		return nil, err
	}
	installedApps := c.listInstalledApps()
	for _, installedApp := range installedApps {
		if installedApp.AppName == tools.SampleApp {
			return &installedApp, nil
		}
	}
	return nil, fmt.Errorf("Sample app not found")

}

func (c *CloudClient) searchForSampleApp() (*tools.AppWithLatestVersion, error) {
	appSearchRequest := tools.AppSearchRequest{
		SearchTerm:         c.appToOperateOn,
		ShowUnofficialApps: c.searchForUnofficialApps,
	}
	responseBody, err := c.parent.DoRequest(tools.AppsSearchPath, appSearchRequest, "")
	if err != nil {
		return nil, err
	}
	var apps []tools.AppWithLatestVersion
	err = json.Unmarshal(responseBody, &apps)
	if err != nil {
		return nil, err
	}
	if len(apps) != 1 {
		return nil, fmt.Errorf("exactly one app expected, but actual number was: %d", len(apps))
	}
	app := apps[0]
	if app.AppName != c.appToOperateOn || app.Maintainer != tools.SampleMaintainer {
		return nil, fmt.Errorf("wrong app found")
	}
	return &app, nil
}

func (c *CloudClient) findVersion(appId, versionName string) *tools.VersionInfo {
	responseBody, err := c.parent.DoRequest(tools.VersionsListPath, tools.NumberString{Value: appId}, "")
	assert.Nil(c.t, err)
	var versions []tools.VersionInfo
	err = json.Unmarshal(responseBody, &versions)
	assert.Nil(c.t, err)
	for _, v := range versions {
		if v.Name == versionName {
			return &v
		}
	}
	c.t.Fatal("No version found")
	return nil
}

func (c *CloudClient) createSampleAppBackup() {
	responseBody, err := c.parent.DoRequest(tools.AppsListPath, nil, "")
	assert.Nil(c.t, err)
	var apps []tools.AppDto
	err = json.Unmarshal(responseBody, &apps)
	assert.Nil(c.t, err)
	if len(apps) == 0 {
		c.t.Fatal("No apps found")
	}
	var sampleApp *tools.AppDto
	for _, app := range apps {
		if app.AppName == "sampleapp" {
			sampleApp = &app
			break
		}
	}
	if sampleApp == nil {
		c.t.Fatal("No sample app found")
	}
	_, err = c.parent.DoRequest(tools.BackupsCreatePath, tools.NumberString{Value: sampleApp.AppId}, "")
	assert.Nil(c.t, err)
}

func (c *CloudClient) listAppBackups(maintainer, appName string, isLocal bool) []tools.BackupInfo {
	backupListRequest := tools.BackupListRequest{
		Maintainer: maintainer,
		AppName:    appName,
		IsLocal:    isLocal,
	}
	responseBody, err := c.parent.DoRequest(tools.BackupsListPath, backupListRequest, "")
	assert.Nil(c.t, err)
	var backups []tools.BackupInfo
	if responseBody == nil {
		return nil
	}
	err = json.Unmarshal(responseBody, &backups)
	assert.Nil(c.t, err)
	return backups
}

func (c *CloudClient) pruneApp(appId string) error {
	_, err := c.parent.DoRequest(tools.AppsPrunePath, tools.NumberString{Value: appId}, "")
	return err
}

func (c *CloudClient) listInstalledApps() []tools.AppDto {
	responseBody, err := c.parent.DoRequest(tools.AppsListPath, nil, "")
	assert.Nil(c.t, err)
	var appDtos []tools.AppDto
	err = json.Unmarshal(responseBody, &appDtos)
	assert.Nil(c.t, err)
	return appDtos
}

func (c *CloudClient) getInstalledSampleApp() *tools.AppDto {
	var sampleApp *tools.AppDto
	apps := c.listInstalledApps()
	for _, app := range apps {
		if app.AppName == tools.SampleApp {
			sampleApp = &app
		}
	}
	return sampleApp
}

func (c *CloudClient) updateApp(appId string) error {
	_, err := c.parent.DoRequest(tools.AppsUpdatePath, tools.NumberString{Value: appId}, "")
	return err
}

func (c *CloudClient) restoreBackup(backupId string, isLocal bool) {
	_, err := c.parent.DoRequest(tools.BackupsRestorePath, tools.BackupOperationRequest{BackupId: backupId, IsLocal: isLocal}, "")
	assert.Nil(c.t, err)
}

func (c *CloudClient) assertContent(expectedContent string) error {
	c.listInstalledApps()

	// Natively running backend can't access any containers and therefore not proxy requests
	if tools.Profile != tools.DOCKER_TEST {
		return nil
	}

	secret, err := c.getSecret()
	if err != nil {
		return err
	}
	client := &http.Client{}
	req, err := http.NewRequest("GET", getAppRequestUrlWithSecret(secret), nil)
	if err != nil {
		logger.Error("Failed to create request: %v", err)
		return err
	}

	req.AddCookie(c.parent.Cookie)

	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Failed to send request: %v", err)
		return err
	}
	defer utils.Close(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read response body: %v", err)
		return err
	}
	actual := string(body)
	if actual != expectedContent {
		return fmt.Errorf("Expected %s but got %s", expectedContent, actual)
	}
	return nil
}

func getAppRequestUrlWithSecret(secret string) string {
	return fmt.Sprintf("http://sampleapp.localhost/api?%s=%s", tools.OcelotQuerySecretName, secret)
}

// Direct calls to this function in test code are limited to security testing. For true app availability testing, please use assertContent.
func (c *CloudClient) assertContentUsingSecret(secret string) error {
	client := &http.Client{}
	req, err := http.NewRequest("GET", getAppRequestUrlWithSecret(secret), nil)
	if err != nil {
		logger.Error("Failed to create request: %v", err)
		return err
	}

	req.AddCookie(c.parent.Cookie)

	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Failed to send request: %v", err)
		return err
	}
	defer utils.Close(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read response body: %v", err)
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New(utils.GetErrMsg(resp.StatusCode, string(body)))
	}
	return nil
}

func (c *CloudClient) setHostValue(host string) {
	_, err := c.parent.DoRequest(tools.SettingsHostSavePath, tools.NumberString{Value: host}, "")
	assert.Nil(c.t, err)
}

func (c *CloudClient) getHostValue() (string, error) {
	responseBody, err := c.parent.DoRequest(tools.SettingsHostReadPath, nil, "")
	if err != nil {
		return "", err
	}
	var hostString tools.HostString
	err = json.Unmarshal(responseBody, &hostString)
	if err != nil {
		return "", err
	}
	return hostString.Value, nil
}

func (c *CloudClient) startApp(appId string) error {
	_, err := c.parent.DoRequest(tools.AppsStartPath, tools.NumberString{Value: appId}, "")
	return err
}

func (c *CloudClient) getUsers() []security.UserDto {
	responseBody, err := c.parent.DoRequest(tools.UsersListPath, nil, "")
	assert.Nil(c.t, err)
	var users []security.UserDto
	err = json.Unmarshal(responseBody, &users)
	assert.Nil(c.t, err)
	return users
}

func (c *CloudClient) createUser(user string, password string) error {
	credentials := security.Credentials{
		Username: user,
		Password: password,
	}
	_, err := c.parent.DoRequest(tools.UsersCreatePath, credentials, "")
	return err
}

func (c *CloudClient) deleteUser(userId string) error {
	_, err := c.parent.DoRequest(tools.UsersDeletePath, tools.NumberString{Value: userId}, "")
	return err
}

func (c *CloudClient) deleteUserIfPresent() {
	users := c.getUsers()
	normalUser, err := c.getNormalUser(users)
	if err == nil {
		assert.Nil(c.t, c.deleteUser(normalUser.Id))
	}
}

func (c *CloudClient) getNormalUser(users []security.UserDto) (*security.UserDto, error) {
	for _, user := range users {
		if user.Name == "user" {
			return &user, nil
		}
	}
	return nil, fmt.Errorf("no normal user found")
}

func (c *CloudClient) createBackup(appId string) {
	_, err := c.parent.DoRequest(tools.BackupsCreatePath, tools.NumberString{Value: appId}, "")
	assert.Nil(c.t, err)
}

func (c *CloudClient) deleteBackup(backupId string, isLocal bool) {
	deleteBackupRequest := tools.BackupOperationRequest{
		BackupId: backupId,
		IsLocal:  isLocal,
	}
	_, err := c.parent.DoRequest(tools.BackupsDeletePath, deleteBackupRequest, "")
	assert.Nil(c.t, err)
}

func (c *CloudClient) readSshConfigs() (*tools.RemoteBackupRepository, error) {
	responseBody, err := c.parent.DoRequest(tools.SettingsSshReadPath, nil, "")
	if err != nil {
		return nil, err
	}
	var remoteRepo tools.RemoteBackupRepository
	err = json.Unmarshal(responseBody, &remoteRepo)
	if err != nil {
		return nil, err
	}
	return &remoteRepo, nil
}

func (c *CloudClient) saveSshConfigs(repository tools.RemoteBackupRepository) {
	_, err := c.parent.DoRequest(tools.SettingsSshSavePath, repository, "")
	assert.Nil(c.t, err)
}

func (c *CloudClient) testSshAccess(repo tools.RemoteBackupRepository) error {
	_, err := c.parent.DoRequest(tools.SettingsSshTestAccessPath, repo, "")
	return err
}

func (c *CloudClient) getKnownHosts(repo tools.RemoteBackupRepository) string {
	responseBody, err := c.parent.DoRequest(tools.SettingsSshKnownHostsPath, repo, "")
	assert.Nil(c.t, err)
	var knownHostsWrapper tools.KnownHostsString
	err = json.Unmarshal(responseBody, &knownHostsWrapper)
	assert.Nil(c.t, err)
	return knownHostsWrapper.Value
}

func (c *CloudClient) checkAuth() tools.Authorization {
	responseBody, err := c.parent.DoRequest(tools.CheckAuthPath, nil, "")
	assert.Nil(c.t, err)
	var auth tools.Authorization
	err = json.Unmarshal(responseBody, &auth)
	assert.Nil(c.t, err)
	return auth
}

func (c *CloudClient) stopApp(appId string) error {
	_, err := c.parent.DoRequest(tools.AppsStopPath, tools.NumberString{Value: appId}, "")
	return err
}

func (c *CloudClient) listAppsInBackupRepo(isLocal bool) ([]tools.MaintainerAndApp, error) {
	responseBody, err := c.parent.DoRequest(tools.BackupsListAppsPath, tools.SingleBool{Value: isLocal}, "")
	if err != nil {
		return nil, err
	}
	var apps []tools.MaintainerAndApp
	err = json.Unmarshal(responseBody, &apps)
	if err != nil {
		return nil, err
	}
	return apps, nil
}

func (c *CloudClient) getMaintenanceSettings() backups.MaintenanceSettings {
	responseBody, err := c.parent.DoRequest(tools.SettingsMaintenanceReadPath, nil, "")
	assert.Nil(c.t, err)
	var maintenanceSettings backups.MaintenanceSettings
	err = json.Unmarshal(responseBody, &maintenanceSettings)
	assert.Nil(c.t, err)
	return maintenanceSettings
}

func (c *CloudClient) setMaintenanceSettings(settings backups.MaintenanceSettings) {
	_, err := c.parent.DoRequest(tools.SettingsMaintenanceSavePath, settings, "")
	assert.Nil(c.t, err)
}

func (c *CloudClient) changePassword(newPassword string) error {
	_, err := c.parent.DoRequest(tools.ChangePasswordPath, tools.PasswordString{Value: newPassword}, "")
	return err
}

func (c *CloudClient) downloadApp() (*store.VersionDownload, error) {
	storeApp, err := c.searchForSampleApp()
	if err != nil {
		return nil, err
	}
	versionInfo := c.findVersion(storeApp.AppId, "2.0")

	responseBody, err := c.parent.DoRequest(tools.VersionsDownloadPath, tools.NumberString{Value: versionInfo.Id}, "")
	if err != nil {
		return nil, err
	}
	var download store.VersionDownload
	err = json.Unmarshal(responseBody, &download)
	if err != nil {
		return nil, err
	}
	return &download, nil
}

func AssertCorsHeaders(t *testing.T, expectedAllowOrigin, expectedAllowMethods, expectedAllowHeaders, expectedAllowCredentials string) {
	cloud := getClientAndLogin(t)
	resp, err := cloud.parent.DoRequestWithFullResponse("/", nil, "")
	assert.Nil(t, err)

	allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	assert.Equal(t, expectedAllowOrigin, allowOrigin)

	allowMethods := resp.Header.Get("Access-Control-Allow-Methods")
	assert.Equal(t, expectedAllowMethods, allowMethods)

	allowHeaders := resp.Header.Get("Access-Control-Allow-Headers")
	assert.Equal(t, expectedAllowHeaders, allowHeaders)

	allowCredentials := resp.Header.Get("Access-Control-Allow-Credentials")
	assert.Equal(t, expectedAllowCredentials, allowCredentials)
}
