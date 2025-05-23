package store

import (
	"github.com/ocelot-cloud/shared/assert"
	"ocelot/backend/apps/common"
	"ocelot/backend/tools"
	"testing"
)

func TestMain(m *testing.M) {
	common.InitializeDatabase(false, false)
	defer common.WipeWholeDatabase()
	m.Run()
}

func TestProvideAppStoreClient(t *testing.T) {
	tools.CreateOcelotTempDir()
	hubClient := ProvideAppStoreClient(INTEGRATION_TEST)
	realClient, ok := hubClient.(*hubClientReal)
	assert.True(t, ok)
	assert.Equal(t, "http://localhost:8082", realClient.client.RootUrl)

	hubClient = ProvideAppStoreClient(PROD)
	realClient, ok = hubClient.(*hubClientReal)
	assert.True(t, ok)
	assert.Equal(t, "https://store.ocelot-cloud.org", realClient.client.RootUrl)

	hubClient = ProvideAppStoreClient(MOCK)
	_, ok = hubClient.(*appStoreClientMock)
	assert.True(t, ok)
}

func TestHubClientMock(t *testing.T) {
	defer common.WipeWholeDatabase()
	var client = ProvideAppStoreClientMock()
	versions, err := client.GetVersions(tools.SampleAppId)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(versions))
	version1 := (versions)[0]
	assert.Equal(t, tools.SampleAppVersion1Name, version1.Name)
	assert.Equal(t, tools.SampleAppVersion1Id, version1.Id)
	assert.Equal(t, tools.SampleAppVersion1CreationTimestamp, version1.VersionCreationTimestamp)

	apps, err := client.SearchApps(getSampleAppSearchRequest())
	assert.Nil(t, err)
	assert.Equal(t, 1, len(apps))
	sampleApp := (apps)[0]
	assert.Equal(t, tools.SampleMaintainer, sampleApp.Maintainer)
	assert.Equal(t, tools.SampleAppId, sampleApp.AppId)
	assert.Equal(t, tools.SampleApp, sampleApp.AppName)
	assert.Equal(t, tools.SampleAppVersion2Id, sampleApp.LatestVersionId)
	assert.Equal(t, tools.SampleAppVersion2Name, sampleApp.LatestVersionName)

	apps, err = client.SearchApps(getSampleAppSearchRequest())
	assert.Nil(t, err)
	sampleApp = (apps)[0]
	assert.Equal(t, tools.SampleMaintainer, sampleApp.Maintainer)
	assert.Equal(t, tools.SampleAppId, sampleApp.AppId)
	assert.Equal(t, tools.SampleApp, sampleApp.AppName)
	assert.Equal(t, tools.SampleAppVersion2Id, sampleApp.LatestVersionId)
	assert.Equal(t, tools.SampleAppVersion2Name, sampleApp.LatestVersionName)

	versions, err = client.GetVersions(tools.SampleAppId)
	assert.Nil(t, err)
	version2 := (versions)[1]
	assert.Equal(t, tools.SampleAppVersion2Name, version2.Name)
	assert.Equal(t, tools.SampleAppVersion2Id, version2.Id)
	assert.Equal(t, tools.SampleAppVersion2CreationTimestamp, version2.VersionCreationTimestamp)

	tag, err := client.DownloadVersion(tools.SampleAppVersion1Id)
	assert.Nil(t, err)
	assert.Equal(t, tools.SampleMaintainer, tag.Maintainer)
	assert.Equal(t, tools.SampleApp, tag.AppName)
	assert.Equal(t, tools.SampleAppVersion1Name, tag.VersionName)
	assert.True(t, len(tag.Content) > 100)
	assert.Equal(t, tools.SampleAppVersion1CreationTimestamp, tag.VersionCreationTimestamp)

	tag, err = client.DownloadVersion(tools.SampleAppVersion2Id)
	assert.Nil(t, err)
	assert.Equal(t, tools.SampleMaintainer, tag.Maintainer)
	assert.Equal(t, tools.SampleApp, tag.AppName)
	assert.Equal(t, tools.SampleAppVersion2Name, tag.VersionName)
	assert.True(t, len(tag.Content) > 100)
	assert.Equal(t, tools.SampleAppVersion2CreationTimestamp, tag.VersionCreationTimestamp)
}

func getSampleAppSearchRequest() tools.AppSearchRequest {
	return tools.AppSearchRequest{
		SearchTerm:         "sample",
		ShowUnofficialApps: true,
	}
}

func TestShowUnofficialAppsInMock(t *testing.T) {
	defer common.WipeWholeDatabase()
	var client = ProvideAppStoreClientMock()
	apps, err := client.SearchApps(getSampleAppSearchRequest())
	assert.Nil(t, err)
	assert.Equal(t, 1, len(apps))

	searchRequestNotShowUnofficialApps := getSampleAppSearchRequest()
	searchRequestNotShowUnofficialApps.ShowUnofficialApps = false
	apps, err = client.SearchApps(searchRequestNotShowUnofficialApps)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(apps))
}
