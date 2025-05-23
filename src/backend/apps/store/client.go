package store

import (
	"fmt"
	"github.com/ocelot-cloud/shared/utils"
	"github.com/ocelot-cloud/shared/validation"
	"ocelot/backend/apps/common"
	"ocelot/backend/tools"
	"strconv"
	"time"
)

type AppStoreClientType int

const (
	PROD AppStoreClientType = iota
	MOCK
	INTEGRATION_TEST
)

func ProvideAppStoreClient(appStoreClientType AppStoreClientType) common.AppStoreClient {
	if appStoreClientType == MOCK {
		return ProvideAppStoreClientMock()
	} else {
		var storeUrl string
		if appStoreClientType == INTEGRATION_TEST {
			storeUrl = "http://localhost:8082"
		} else {
			storeUrl = "https://store.ocelot-cloud.org"
		}
		client := utils.ComponentClient{
			RootUrl: storeUrl,
		}
		return &hubClientReal{&client}
	}
}

type hubClientReal struct {
	client *utils.ComponentClient
}

func (h hubClientReal) SearchApps(searchRequest tools.AppSearchRequest) ([]tools.AppWithLatestVersion, error) {
	responseBody, err := h.client.DoRequest("/api/apps/search", searchRequest, "")
	if err != nil {
		Logger.Error("Failed to search apps: %v", err)
		return nil, err
	}
	userAndAppList, err := utils.UnpackResponse[[]tools.AppWithLatestVersion](responseBody)
	if err != nil {
		Logger.Error("Failed to unpack response: %v", err)
		return nil, err
	}
	return *userAndAppList, nil
}

type StringTag struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

var CreationDateOfMockedVersion = time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

func (h hubClientReal) GetVersions(appId string) ([]tools.VersionInfo, error) {
	responseBody, err := h.client.DoRequest("/api/versions/list", tools.NumberString{Value: appId}, "")
	if err != nil {
		Logger.Error("Failed to get versions: %v", err)
		return nil, err
	}

	stringTagList, err := utils.UnpackResponse[[]StringTag](responseBody)
	if err != nil {
		Logger.Error("Failed to unpack response: %v", err)
		return nil, err
	}

	var versions []tools.VersionInfo
	for _, stringTag := range *stringTagList {
		tagId, err := strconv.Atoi(stringTag.Id)
		if err != nil {
			Logger.Error("Failed to convert tag id to int: %v", err)
			return nil, err
		}
		versions = append(versions, tools.VersionInfo{Id: strconv.Itoa(tagId), Name: stringTag.Name, VersionCreationTimestamp: CreationDateOfMockedVersion})
	}

	return versions, nil
}

func (h hubClientReal) DownloadVersion(tagId string) (*tools.FullVersionInfo, error) {
	result, err := h.client.DoRequest("/api/versions/download", tools.NumberString{Value: tagId}, "")
	if err != nil {
		Logger.Error("Failed to download version: %v", err)
		return nil, err
	}
	fullTagInfo, err := utils.UnpackResponse[tools.FullVersionInfo](result)
	if err != nil {
		Logger.Error("Failed to unpack response: %v", err)
		return nil, err
	}
	err = validation.ValidateVersion(fullTagInfo.Content, fullTagInfo.Maintainer, fullTagInfo.AppName)
	if err != nil {
		Logger.Error("Failed to validate version: %v", err)
		return nil, err
	}
	return fullTagInfo, nil
}

type appStoreClientMock struct {
	apps     []tools.AppWithLatestVersion
	versions []tools.FullVersionInfo
}

func ProvideAppStoreClientMock() common.AppStoreClient {
	client := appStoreClientMock{
		apps:     nil,
		versions: nil,
	}

	client.addSampleAppWithFirstVersion()
	client.addSampleAppWithSecondVersion()
	return &client
}

func (h *appStoreClientMock) addOrUpdateApp(
	appId,
	appName,
	maintainer,
	latestVersionId,
	latestVersionName string,
	content []byte,
) {
	var found bool
	for i := range h.apps {
		if h.apps[i].AppName == appName {
			found = true
			h.apps[i].LatestVersionId = latestVersionId
			h.apps[i].LatestVersionName = latestVersionName
		}
	}
	if !found {
		h.apps = append(h.apps, tools.AppWithLatestVersion{
			Maintainer:        maintainer,
			AppId:             appId,
			AppName:           appName,
			LatestVersionId:   latestVersionId,
			LatestVersionName: latestVersionName,
		})
	}

	h.versions = append(h.versions, tools.FullVersionInfo{
		Maintainer:               maintainer,
		AppName:                  appName,
		VersionName:              latestVersionName,
		Content:                  content,
		VersionCreationTimestamp: tools.SampleAppCreationTimestamp.Add(-time.Hour),
	})
}

func (h *appStoreClientMock) addSampleAppWithFirstVersion() {
	sampleAppContent := tools.GetSampleAppContent()
	h.addOrUpdateApp(
		tools.SampleAppId,
		tools.SampleApp,
		tools.SampleMaintainer,
		tools.SampleAppVersion1Id,
		tools.SampleAppVersion1Name,
		sampleAppContent,
	)
}

func (h *appStoreClientMock) addSampleAppWithSecondVersion() {
	sampleAppContent := tools.CreateTempZippedApp(
		"/sampleapp",
		"docker-compose-updated.yml",
		true,
		tools.SampleAppDir,
		"app.yml",
	)

	for i := range h.apps {
		if h.apps[i].AppName == "sampleapp" {
			h.apps[i].LatestVersionId = tools.SampleAppVersion2Id
			h.apps[i].LatestVersionName = tools.SampleAppVersion2Name
		}
	}
	h.versions = append(h.versions, tools.FullVersionInfo{
		Maintainer:               tools.SampleMaintainer,
		AppName:                  tools.SampleApp,
		VersionName:              tools.SampleAppVersion2Name,
		Content:                  sampleAppContent,
		VersionCreationTimestamp: tools.SampleAppCreationTimestamp.Add(+time.Hour),
	})
}

func (h *appStoreClientMock) SearchApps(searchRequest tools.AppSearchRequest) ([]tools.AppWithLatestVersion, error) {
	if searchRequest.ShowUnofficialApps {
		return h.apps, nil
	} else {
		return []tools.AppWithLatestVersion{}, nil
	}
}

func (h *appStoreClientMock) GetVersions(appId string) ([]tools.VersionInfo, error) {
	version1 := tools.VersionInfo{Id: tools.SampleAppVersion1Id, Name: tools.SampleAppVersion1Name, VersionCreationTimestamp: tools.SampleAppVersion1CreationTimestamp}
	version2 := tools.VersionInfo{Id: tools.SampleAppVersion2Id, Name: tools.SampleAppVersion2Name, VersionCreationTimestamp: tools.SampleAppVersion2CreationTimestamp}
	return []tools.VersionInfo{version1, version2}, nil
}

func (h *appStoreClientMock) DownloadVersion(versionId string) (*tools.FullVersionInfo, error) {
	var wantedVersionName string
	if versionId == tools.SampleAppVersion1Id {
		wantedVersionName = tools.SampleAppVersion1Name
	} else if versionId == tools.SampleAppVersion2Id {
		wantedVersionName = tools.SampleAppVersion2Name
	} else {
		Logger.Error("Version id not allowed: %s", versionId)
		return nil, fmt.Errorf("version id not allowed")
	}

	for _, version := range h.versions {
		if version.VersionName == wantedVersionName {
			return &version, nil
		}
	}
	Logger.Error("version id not found: %s", versionId)
	return nil, fmt.Errorf("version id not found")
}
