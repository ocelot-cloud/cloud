//go:build fast

package common

import (
	"github.com/ocelot-cloud/shared/assert"
	"ocelot/backend/tools"
	"testing"
)

func TestMain(t *testing.M) {
	initializeDatabase(false, false)
	WipeWholeDatabase()
	defer WipeWholeDatabase()
	t.Run()
}

func TestAppCreation(t *testing.T) {
	tools.CreateOcelotTempDir()
	defer WipeWholeDatabase()
	appId := createSampleAppAndReturnRepoId(t)

	app, err := AppRepo.GetApp(appId)
	assert.Nil(t, err)
	assertAppFields(t, *app)
}

func assertAppFields(t *testing.T, app tools.RepoApp) {
	assert.Equal(t, tools.SampleMaintainer, app.Maintainer)
	assert.Equal(t, tools.SampleApp, app.AppName)
	assert.Equal(t, tools.SampleAppVersion1Name, app.VersionName)
	assert.Equal(t, tools.SampleAppVersion1CreationTimestamp, app.VersionCreationTimestamp)
	assert.Equal(t, tools.GetSampleAppContent(), app.VersionContent)
	assert.False(t, app.ShouldBeRunning)
}

func TestDeleteApp(t *testing.T) {
	defer WipeWholeDatabase()
	notExistingId := 12345
	assert.Nil(t, AppRepo.DeleteApp(notExistingId))

	apps, err := AppRepo.ListApps()
	assert.Nil(t, err)
	initialNumberOfApps := len(apps)

	appId := createSampleAppAndReturnRepoId(t)
	apps, err = AppRepo.ListApps()
	assert.Nil(t, err)
	assert.Equal(t, initialNumberOfApps+1, len(apps))

	assert.Nil(t, AppRepo.DeleteApp(appId))

	_, err = AppRepo.GetApp(appId)
	assert.NotNil(t, err)

	apps, err = AppRepo.ListApps()
	assert.Nil(t, err)
	assert.Equal(t, initialNumberOfApps, len(apps))

	_, err = AppRepo.GetAppId(tools.SampleMaintainer, tools.SampleApp)
	assert.NotNil(t, err)
}

func TestCreatingAppTwoTimes(t *testing.T) {
	defer WipeWholeDatabase()
	assert.Nil(t, CreateSampleAppInRepo())
	err := CreateSampleAppInRepo()
	assert.NotNil(t, err)
	assert.Equal(t, "app already exists", err.Error())
}

func createSampleAppAndReturnRepoId(t *testing.T) int {
	assert.Nil(t, CreateSampleAppInRepo())
	appId, err := AppRepo.GetAppId(tools.SampleMaintainer, tools.SampleApp)
	assert.Nil(t, err)
	return appId
}

func TestSetShouldBeRunning(t *testing.T) {
	defer WipeWholeDatabase()
	appId := createSampleAppAndReturnRepoId(t)

	app, err := AppRepo.GetApp(appId)
	assert.Nil(t, err)
	assert.False(t, app.ShouldBeRunning)

	assert.Nil(t, AppRepo.SetAppShouldBeRunning(appId, true))
	app, err = AppRepo.GetApp(appId)
	assert.Nil(t, err)
	assert.True(t, app.ShouldBeRunning)
}

func TestUpdateVersion(t *testing.T) {
	defer WipeWholeDatabase()
	appId := createSampleAppAndReturnRepoId(t)

	app, err := AppRepo.GetApp(appId)
	assert.Nil(t, err)
	assert.Equal(t, tools.SampleAppVersion1Name, app.VersionName)
	assert.Equal(t, tools.SampleAppVersion1CreationTimestamp, app.VersionCreationTimestamp)
	assert.Equal(t, tools.GetSampleAppContent(), app.VersionContent)

	newContent := []byte("new content")
	version2 := tools.VersionMetaData{
		Name:              tools.SampleAppVersion2Name,
		CreationTimestamp: tools.SampleAppVersion2CreationTimestamp,
		Content:           newContent,
	}
	assert.Nil(t, AppRepo.UpdateVersion(appId, version2))

	app, err = AppRepo.GetApp(appId)
	assert.Nil(t, err)
	assert.Equal(t, tools.SampleAppVersion2Name, app.VersionName)
	assert.Equal(t, tools.SampleAppVersion2CreationTimestamp, app.VersionCreationTimestamp)
	assert.Equal(t, newContent, app.VersionContent)
}

func TestDoesAppExist(t *testing.T) {
	defer WipeWholeDatabase()

	assert.False(t, AppRepo.DoesAppExist(tools.SampleMaintainer, tools.SampleApp))
	_ = createSampleAppAndReturnRepoId(t)
	assert.True(t, AppRepo.DoesAppExist(tools.SampleMaintainer, tools.SampleApp))
}
