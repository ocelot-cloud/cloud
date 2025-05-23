package store

import (
	"errors"
	"fmt"
	"github.com/ocelot-cloud/shared/utils"
	"github.com/ocelot-cloud/shared/validation"
	"net/http"
	"ocelot/backend/apps/common"
	"ocelot/backend/tools"
)

var Logger = tools.Logger

func GetVersionsHandler(w http.ResponseWriter, r *http.Request) {
	appIdString, err := validation.ReadBody[tools.NumberString](w, r)
	if err != nil {
		return
	}

	versions, err := common.StoreClient.GetVersions(appIdString.Value)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	utils.SendJsonResponse(w, versions)
}

func AppSearchHandler(w http.ResponseWriter, r *http.Request) {
	appSearchRequest, err := validation.ReadBody[tools.AppSearchRequest](w, r)
	if err != nil {
		return
	}

	apps, err := common.StoreClient.SearchApps(*appSearchRequest)
	if err != nil {
		Logger.Info("Failed to search apps: %v", err)
		http.Error(w, "failed to search apps", http.StatusBadRequest)
		return
	}
	if len(apps) == 0 {
		Logger.Info("no apps found")
		http.Error(w, "no apps found", http.StatusNotFound)
		return
	}

	utils.SendJsonResponse(w, apps)
}

func VersionInstallationHandler(w http.ResponseWriter, r *http.Request) {
	repoApp, err := downloadAppFromStoreAndCheckWhetherItAlreadyExistence(w, r)
	if err != nil {
		return
	}

	err = common.UpsertApp(*repoApp)
	if err != nil {
		Logger.Info("failed to upsert app: %v", err)
		http.Error(w, "failed to upsert app", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func downloadAppFromStoreAndCheckWhetherItAlreadyExistence(w http.ResponseWriter, r *http.Request) (*tools.RepoApp, error) {
	versionIdString, err := validation.ReadBody[tools.NumberString](w, r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, errors.New("")
	}

	repoApp, err := common.DownloadTag(versionIdString.Value)
	if err != nil {
		Logger.Info("failed to download version: %v")
		http.Error(w, "failed to download version", http.StatusBadRequest)
		return nil, errors.New("")
	}

	if common.AppRepo.DoesAppExist(repoApp.Maintainer, repoApp.AppName) {
		msg := fmt.Sprintf("can't install app '%s / %s' because it is already installed", repoApp.Maintainer, repoApp.AppName)
		Logger.Info(msg)
		http.Error(w, msg, http.StatusConflict)
		return nil, errors.New("")
	}
	return repoApp, nil
}

type VersionDownload struct {
	FileName string `json:"file_name"`
	Content  []byte `json:"content"`
}

func VersionDownloadHandler(w http.ResponseWriter, r *http.Request) {
	repoApp, err := downloadAppFromStoreAndCheckWhetherItAlreadyExistence(w, r)
	if err != nil {
		return
	}

	versionDownload := VersionDownload{
		FileName: repoApp.VersionName + ".zip",
		Content:  repoApp.VersionContent,
	}
	utils.SendJsonResponse(w, versionDownload)
}
