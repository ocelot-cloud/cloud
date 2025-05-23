//go:build integration

package store

import (
	"github.com/ocelot-cloud/shared/assert"
	"ocelot/backend/tools"
	"testing"
	"time"
)

func TestStoreClientReal(t *testing.T) {
	sampleMaintainer := "sampleuser"
	sampleAppName := "nginx"
	sampleVersionName := "0.0.1"

	hub := ProvideAppStoreClient(INTEGRATION_TEST)
	apps, err := hub.SearchApps(getSampleAppSearchRequest())
	assert.Nil(t, err)
	assert.Equal(t, 1, len(apps))
	app := (apps)[0]
	assert.Equal(t, sampleMaintainer, app.Maintainer)
	assert.Equal(t, sampleAppName, app.AppName)
	assert.Equal(t, sampleVersionName, app.LatestVersionName)

	versions, err := hub.GetVersions(app.AppId)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(versions))
	tag := (versions)[0]
	assert.Equal(t, sampleVersionName, tag.Name)
	assert.NotEqual(t, "", tag.Id)

	downloadedVersion, err := hub.DownloadVersion(tag.Id)
	assert.Nil(t, err)
	assert.Equal(t, sampleMaintainer, downloadedVersion.Maintainer)
	assert.Equal(t, sampleAppName, downloadedVersion.AppName)
	assert.True(t, len(downloadedVersion.Content) > 100)
	assert.Equal(t, sampleVersionName, downloadedVersion.VersionName)
	assert.True(t, time.Now().UTC().Add(-1*time.Hour).Before(downloadedVersion.VersionCreationTimestamp))
	assert.True(t, time.Now().UTC().Add(1*time.Hour).After(downloadedVersion.VersionCreationTimestamp))
}

func TestStoreClientRealForMaliciousAppsBeingRejected(t *testing.T) {
	sampleMaintainer := "maliciousmaintainer"
	sampleAppName := "maliciousapp"
	sampleVersionName := "0.0.1"

	hub := ProvideAppStoreClient(INTEGRATION_TEST)
	apps, err := hub.SearchApps(tools.AppSearchRequest{
		SearchTerm:         "malicious",
		ShowUnofficialApps: true,
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(apps))
	app := (apps)[0]
	assert.Equal(t, sampleMaintainer, app.Maintainer)
	assert.Equal(t, sampleAppName, app.AppName)
	assert.Equal(t, sampleVersionName, app.LatestVersionName)

	versions, err := hub.GetVersions(app.AppId)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(versions))
	tag := (versions)[0]
	assert.Equal(t, sampleVersionName, tag.Name)
	assert.NotEqual(t, "", tag.Id)

	_, err = hub.DownloadVersion(tag.Id)
	assert.NotNil(t, err)
	assert.Equal(t, "host directories are mounted in service 'maliciousapp' which is forbidden", err.Error())
}

func TestShowUnofficialAppsInIntegration(t *testing.T) {
	hub := ProvideAppStoreClient(INTEGRATION_TEST)
	apps, err := hub.SearchApps(getSampleAppSearchRequest())
	assert.Nil(t, err)
	assert.Equal(t, 1, len(apps))

	onlyShowOfficialAppsSearchRequest := getSampleAppSearchRequest()
	onlyShowOfficialAppsSearchRequest.ShowUnofficialApps = false
	apps, err = hub.SearchApps(onlyShowOfficialAppsSearchRequest)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(apps))
}
