package common

import (
	"ocelot/backend/tools"
)

var Logger = tools.Logger

type AppStoreClient interface {
	SearchApps(searchRequest tools.AppSearchRequest) ([]tools.AppWithLatestVersion, error)
	GetVersions(appId string) ([]tools.VersionInfo, error)
	DownloadVersion(tagId string) (*tools.FullVersionInfo, error)
}

var StoreClient AppStoreClient

func DownloadTag(tagId string) (*tools.RepoApp, error) {
	fullTagInfo, err := StoreClient.DownloadVersion(tagId)
	if err != nil {
		return nil, err
	}
	appInfo := &tools.RepoApp{
		Maintainer:               fullTagInfo.Maintainer,
		AppName:                  fullTagInfo.AppName,
		VersionName:              fullTagInfo.VersionName,
		VersionCreationTimestamp: fullTagInfo.VersionCreationTimestamp,
		VersionContent:           fullTagInfo.Content,
	}
	return appInfo, nil
}

func UpsertApp(app tools.RepoApp) error {
	if AppRepo.DoesAppExist(app.Maintainer, app.AppName) {
		localAppId, err := AppRepo.GetAppId(app.Maintainer, app.AppName)
		if err != nil {
			return err
		}
		versionMetaData := tools.VersionMetaData{
			Name:              app.VersionName,
			CreationTimestamp: app.VersionCreationTimestamp,
			Content:           app.VersionContent,
		}
		err = AppRepo.UpdateVersion(localAppId, versionMetaData)
		if err != nil {
			return err
		}
	} else {
		appInfo := tools.RepoApp{
			Maintainer:               app.Maintainer,
			AppName:                  app.AppName,
			VersionName:              app.VersionName,
			VersionCreationTimestamp: app.VersionCreationTimestamp,
			VersionContent:           app.VersionContent,
			ShouldBeRunning:          false,
		}

		err := AppRepo.CreateApp(appInfo)
		if err != nil {
			return err
		}
	}
	return nil
}

func CreateSampleAppInRepo() error {
	return AppRepo.CreateApp(GetSampleAppInfo())
}

func GetSampleAppInfo() tools.RepoApp {
	return tools.RepoApp{
		Maintainer:               tools.SampleMaintainer,
		AppName:                  tools.SampleApp,
		VersionName:              tools.SampleAppVersion1Name,
		VersionCreationTimestamp: tools.SampleAppVersion1CreationTimestamp,
		VersionContent:           tools.GetSampleAppContent(),
		ShouldBeRunning:          false,
	}
}
