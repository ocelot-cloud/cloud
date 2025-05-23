package setup

import (
	"github.com/ocelot-cloud/shared/utils"
	"net/http"
	"ocelot/backend/apps/common"
	"ocelot/backend/certs"
	"ocelot/backend/clients"
	"ocelot/backend/security"
	"ocelot/backend/tools"
)

var logger = tools.Logger

func InitializeApplication() {
	security.RegisterRoutes([]security.Route{
		{Path: tools.LoginPath, HandlerFunc: loginHandler, AccessLevel: security.Anonymous},
		{Path: tools.CheckAuthPath, HandlerFunc: security.CheckAuthHandler, AccessLevel: security.Anonymous},
		{Path: tools.SecretPath, HandlerFunc: SecretHandler, AccessLevel: security.User},
		{Path: tools.SettingsCertificateUploadPath, HandlerFunc: certs.CertificateUploadHandler, AccessLevel: security.Admin},
	})

	if tools.Config.IsGuiEnabled {
		initializeFrontendResourceDelivery()
	}

	var handler http.Handler = http.HandlerFunc(ApplyAuthMiddleware)
	if tools.Config.AreCrossOriginRequestsAllowed {
		handler = utils.GetCorsDisablingHandler(handler)
	}

	startAppsThatShouldBeRunning()
	certs.StartServers(handler)
}

func startAppsThatShouldBeRunning() {
	apps, err := common.AppRepo.ListApps()
	if err != nil {
		Logger.Error("Could not get running app ids: %v", err)
	}

	var idsOfRunningApps []int
	for _, app := range apps {
		if app.ShouldBeRunning {
			idsOfRunningApps = append(idsOfRunningApps, app.AppId)
		}
	}

	for _, appId := range idsOfRunningApps {
		err = clients.Apps.StartApp(appId)
		if err != nil {
			Logger.Warn("Could not start app with id %d: %v", appId, err)
		}
	}
}

func initializeFrontendResourceDelivery() {
	tools.Router.PathPrefix("/").Handler(http.HandlerFunc(tools.FrontendHandlerFunc))
}
