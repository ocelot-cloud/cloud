package clients

import (
	"net/http"
	"ocelot/backend/apps/common"
	"ocelot/backend/tools"
)

var Apps AppManager

type AppManager interface {
	StartApp(appId int) error
	StopApp(appId int) error
	ProxyRequestToTheAppsDockerContainer(w http.ResponseWriter, r *http.Request)
}

type MockAppManager struct{}

func (m *MockAppManager) StartApp(appId int) error {
	_, err := common.AppRepo.GetApp(appId)
	if err != nil {
		return err
	}
	err = common.AppRepo.SetAppShouldBeRunning(appId, true)
	if err != nil {
		return err
	}
	return nil
}

func (m *MockAppManager) StopApp(appId int) error {
	_, err := common.AppRepo.GetApp(appId)
	if err != nil {
		return err
	}
	err = common.AppRepo.SetAppShouldBeRunning(appId, false)
	if err != nil {
		return err
	}
	return nil
}

func (m *MockAppManager) ProxyRequestToTheAppsDockerContainer(w http.ResponseWriter, r *http.Request) {
	appId, err := common.AppRepo.GetAppId(tools.SampleMaintainer, tools.SampleApp)
	if err != nil {
		http.Error(w, "app not available", http.StatusBadRequest)
		return
	}

	app, err := common.AppRepo.GetApp(appId)
	if err != nil {
		http.Error(w, "app not available", http.StatusBadRequest)
		return
	}

	if app.ShouldBeRunning {
		tools.WriteResponse(w, "this is version "+app.VersionName)
	} else {
		http.Error(w, "app not available", http.StatusBadRequest)
	}
}
