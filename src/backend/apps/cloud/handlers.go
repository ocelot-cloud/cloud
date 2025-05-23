package cloud

import (
	"github.com/ocelot-cloud/shared/utils"
	"github.com/ocelot-cloud/shared/validation"
	"net/http"
	"ocelot/backend/apps/common"
	"ocelot/backend/clients"
	"ocelot/backend/security"
	"ocelot/backend/tools"
	"strconv"
)

var Logger = tools.Logger

func AppListHandler(w http.ResponseWriter, r *http.Request) {
	Logger.Debug("app list handler called")
	repoApps, err := common.AppRepo.ListApps()
	if err != nil {
		Logger.Error("Failed to list apps: %v", err)
		return
	}
	appDtos := convertToAppDtos(repoApps)

	context, err := security.GetAuthFromContext(w, r)
	if err != nil {
		return
	}

	if !context.IsAdmin {
		var filteredAppDtos []tools.AppDto
		for _, appDto := range appDtos {
			if appDto.Status == "Available" && !common.IsOcelotDbDto(appDto) {
				filteredAppDtos = append(filteredAppDtos, appDto)
			}
		}
		utils.SendJsonResponse(w, filteredAppDtos)
	} else {
		utils.SendJsonResponse(w, appDtos)
	}
}

func convertToAppDtos(apps []tools.RepoApp) []tools.AppDto {
	var appDtos []tools.AppDto
	for _, app := range apps {
		appConfig, _ := GetAppConfig(app.AppName)
		appDto := tools.AppDto{
			Maintainer:  app.Maintainer,
			AppName:     app.AppName,
			VersionName: app.VersionName,
			AppId:       strconv.Itoa(app.AppId),
			UrlPath:     appConfig.UrlPath,
			Status:      getStatus(app.ShouldBeRunning, true),
		}
		appDtos = append(appDtos, appDto)
	}
	return appDtos
}

func AppStartHandler(w http.ResponseWriter, r *http.Request) {
	err := tools.TryLockAndRespondForError(w, "start app")
	if err != nil {
		return
	}
	defer tools.AppOperationMutex.Unlock()

	Logger.Debug("Starting app")
	appIdString, err := validation.ReadBody[tools.NumberString](w, r)
	if err != nil {
		return
	}

	appId, err := strconv.Atoi(appIdString.Value)
	if err != nil {
		Logger.Error("Failed to convert app id: %v", err)
		http.Error(w, "Failed to convert app id", http.StatusBadRequest)
		return
	}

	if IsOcelotDbApp(w, appId) {
		return
	}

	err = clients.Apps.StartApp(appId)
	if err != nil {
		Logger.Error("Failed to start app: %v", err)
		http.Error(w, "Failed to start app", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func IsOcelotDbApp(w http.ResponseWriter, appId int) bool {
	app, err := common.AppRepo.GetApp(appId)
	if err != nil {
		Logger.Error("Failed to get app: %v", err)
		http.Error(w, "Failed to get app", http.StatusBadRequest)
		return true
	}
	if common.IsOcelotDbApp(*app) {
		Logger.Error("operation not allowed on postgres app")
		http.Error(w, "operation not allowed on postgres app", http.StatusUnauthorized)
		return true
	}
	return false
}

func AppStopHandler(w http.ResponseWriter, r *http.Request) {
	appIdString, err := validation.ReadBody[tools.NumberString](w, r)
	if err != nil {
		return
	}

	appId, err := strconv.Atoi(appIdString.Value)
	if err != nil {
		Logger.Info("Failed to convert app id: %v", err)
		http.Error(w, "Failed to convert app id", http.StatusBadRequest)
		return
	}
	err = tools.TryLockAndRespondForError(w, "stop app")
	if err != nil {
		return
	}
	defer tools.AppOperationMutex.Unlock()

	if IsOcelotDbApp(w, appId) {
		return
	}

	err = clients.Apps.StopApp(appId)
	if err != nil {
		Logger.Info("Failed to stop app: %v", err)
		http.Error(w, "Failed to stop app", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}
