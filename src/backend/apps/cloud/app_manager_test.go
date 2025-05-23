//go:build slow

package cloud

import (
	"github.com/ocelot-cloud/shared/assert"
	"github.com/ocelot-cloud/shared/utils"
	"ocelot/backend/apps/common"
	"ocelot/backend/apps/store"
	"ocelot/backend/clients"
	"ocelot/backend/tools"
	"os"
	"os/exec"
	"testing"
)

func TestMain(m *testing.M) {
	common.InitializeDatabase(false, false)
	err := os.MkdirAll(tools.OcelotCloudTempDir, 0700)
	if err != nil {
		panic(err)
	}
	clients.Apps = GetAppManager()
	defer common.WipeWholeDatabase()
	m.Run()
}

func TestDownloadAndStartApp(t *testing.T) {
	common.WipeWholeDatabase()
	defer common.WipeWholeDatabase()
	common.StoreClient = store.ProvideAppStoreClient(store.MOCK)

	downloadedRepoApp, err := common.DownloadTag(tools.SampleAppVersion1Id)
	assert.Nil(t, err)
	err = common.UpsertApp(*downloadedRepoApp)
	assert.Nil(t, err)

	appId, err := common.AppRepo.GetAppId(tools.SampleMaintainer, tools.SampleApp)
	assert.Nil(t, err)
	app, err := common.AppRepo.GetApp(appId)
	assert.Nil(t, err)
	assert.Equal(t, tools.SampleMaintainer, app.Maintainer)
	assert.Equal(t, tools.SampleApp, app.AppName)
	assert.True(t, len(app.VersionContent) > 100)

	err = clients.Apps.StartApp(appId)
	assert.Nil(t, err)

	err = exec.Command("/bin/sh", "-c", "docker ps | grep -q "+tools.SampleAppDockerContainer).Run()
	assert.Nil(t, err)

	assertCantHaveTwoAppsRunningWithSameNameFromOtherMaintainer(t)

	err = clients.Apps.StopApp(appId)
	assert.Nil(t, err)

	err = exec.Command("/bin/sh", "-c", "docker ps -a | grep -q "+tools.SampleAppDockerContainer).Run()
	assert.NotNil(t, err)
}

func assertCantHaveTwoAppsRunningWithSameNameFromOtherMaintainer(t *testing.T) {
	sampleMaintainer2 := "samplemaintainer2"
	appInfo := common.GetSampleAppInfo()
	appInfo.Maintainer = sampleMaintainer2
	assert.Nil(t, common.AppRepo.CreateApp(appInfo))
	appId2, err := common.AppRepo.GetAppId(sampleMaintainer2, tools.SampleApp)
	assert.Nil(t, err)
	err = clients.Apps.StartApp(appId2)
	assert.NotNil(t, err)
	assert.Equal(t, cantHaveTwoAppsWithSameNameRunningAtSameTime, err.Error())
	assert.Nil(t, common.AppRepo.DeleteApp(appId2))
}

func TestGetStatus(t *testing.T) {
	assert.Equal(t, "Uninitialized", getStatus(false, true))
	assert.Equal(t, "Uninitialized", getStatus(false, false))
	assert.Equal(t, "Available", getStatus(true, true))
	assert.Equal(t, "Starting", getStatus(true, false))
}

func TestGetConfigsFromRepo(t *testing.T) {
	defer common.WipeWholeDatabase()
	UpdateAppConfigs()
	assert.Equal(t, 0, len(appConfigs))

	assert.Nil(t, common.CreateSampleAppInRepo())
	appId, err := common.AppRepo.GetAppId(tools.SampleMaintainer, tools.SampleApp)
	assert.Nil(t, err)
	assert.Nil(t, common.AppRepo.SetAppShouldBeRunning(appId, true))

	UpdateAppConfigs()
	assert.Equal(t, 1, len(appConfigs))
	assert.Equal(t, 3000, appConfigs[tools.SampleApp].Port)
	assert.Equal(t, "/api", appConfigs[tools.SampleApp].UrlPath)
}

func TestStartingAndStoppingApp(t *testing.T) {
	defer common.WipeWholeDatabase()
	appId := SetUpAndStartSampleApp(t)
	shouldContainerBeRunning(t, true)
	ExpectDockerObject(t, Network, true)
	ExpectDockerObject(t, Volume, true)
	ExpectDockerObject(t, Container, true)
	ExpectDockerObject(t, ComposeStack, true)

	assert.Nil(t, clients.Apps.StopApp(appId))
	shouldContainerBeRunning(t, false)
	ExpectDockerObject(t, Network, true)
	ExpectDockerObject(t, Volume, true)
	ExpectDockerObject(t, Container, false)
	ExpectDockerObject(t, ComposeStack, false)
	_, err := common.AppRepo.GetAppId(tools.SampleMaintainer, tools.SampleApp)
	assert.Nil(t, err)
}

func TestIsAnotherAppWithSameAppRunningAtSameTime(t *testing.T) {
	defer common.WipeWholeDatabase()
	maintainer2 := "maintainer2"
	sampleApp := common.GetSampleAppInfo()
	assert.Nil(t, common.AppRepo.CreateApp(sampleApp))
	sampleApp2 := sampleApp
	sampleApp2.Maintainer = maintainer2
	assert.Nil(t, common.AppRepo.CreateApp(sampleApp2))

	isAnotherAppRunning, err := isAnotherAppWithSameNameRunningAtSameTime(&sampleApp)
	assert.Nil(t, err)
	assert.False(t, isAnotherAppRunning)
	isAnotherAppRunning, err = isAnotherAppWithSameNameRunningAtSameTime(&sampleApp2)
	assert.Nil(t, err)
	assert.False(t, isAnotherAppRunning)

	sampleAppId, err := common.AppRepo.GetAppId(sampleApp.Maintainer, sampleApp.AppName)
	assert.Nil(t, err)
	sampleAppId2, err := common.AppRepo.GetAppId(maintainer2, sampleApp.AppName)
	assert.Nil(t, err)
	assert.Nil(t, common.AppRepo.SetAppShouldBeRunning(sampleAppId, true))

	sampleAppUpdated, err := common.AppRepo.GetApp(sampleAppId)
	assert.Nil(t, err)
	sampleApp2Updated, err := common.AppRepo.GetApp(sampleAppId2)
	assert.Nil(t, err)

	isAnotherAppRunning, err = isAnotherAppWithSameNameRunningAtSameTime(sampleAppUpdated)
	assert.Nil(t, err)
	assert.False(t, isAnotherAppRunning)
	isAnotherAppRunning, err = isAnotherAppWithSameNameRunningAtSameTime(sampleApp2Updated)
	assert.Nil(t, err)
	assert.True(t, isAnotherAppRunning)
}

func TestFilledAppYamlFileReading(t *testing.T) {
	backendDir := tools.FindDirWithIdeSupport("src")
	appsCloudDir := backendDir + "/backend/apps/cloud"
	config, err := readAppConfig(appsCloudDir)
	assert.Nil(t, err)

	assert.Equal(t, "/sample", config.UrlPath)
	assert.Equal(t, 1234, config.Port)
}

func TestNonExistingAndEmptyAppYamlFileReading(t *testing.T) {
	tempDir, err := os.MkdirTemp("/tmp", "testing")
	assert.Nil(t, err)
	defer utils.RemoveDir(tempDir)

	config, err := readAppConfig(tempDir)
	assert.Nil(t, err)
	assert.Equal(t, "/", config.UrlPath)
	assert.Equal(t, 80, config.Port)

	_, err = os.Create(tempDir + "/app.yaml")
	assert.Nil(t, err)
	assert.Equal(t, "/", config.UrlPath)
	assert.Equal(t, 80, config.Port)
}
