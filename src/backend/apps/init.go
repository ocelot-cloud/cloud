package apps

import (
	"ocelot/backend/apps/backups"
	"ocelot/backend/apps/cloud"
	"ocelot/backend/apps/common"
	"ocelot/backend/apps/store"
	"ocelot/backend/security"
	"ocelot/backend/tools"
)

func InitializeAppsModule() {
	if tools.Config.UseRealAppStoreClient {
		common.StoreClient = store.ProvideAppStoreClient(store.PROD)
	} else {
		common.StoreClient = store.ProvideAppStoreClient(store.MOCK)
	}

	routes := []security.Route{
		{Path: tools.AppsListPath, HandlerFunc: cloud.AppListHandler, AccessLevel: security.User},
		{Path: tools.AppsStartPath, HandlerFunc: cloud.AppStartHandler, AccessLevel: security.Admin},
		{Path: tools.AppsStopPath, HandlerFunc: cloud.AppStopHandler, AccessLevel: security.Admin},
		{Path: tools.AppsPrunePath, HandlerFunc: backups.AppPruneHandler, AccessLevel: security.Admin},
		{Path: tools.AppsUpdatePath, HandlerFunc: backups.VersionUpdateHandler, AccessLevel: security.Admin},

		{Path: tools.AppsSearchPath, HandlerFunc: store.AppSearchHandler, AccessLevel: security.Admin},
		{Path: tools.VersionsInstallPath, HandlerFunc: store.VersionInstallationHandler, AccessLevel: security.Admin},
		{Path: tools.VersionsDownloadPath, HandlerFunc: store.VersionDownloadHandler, AccessLevel: security.Admin},
		{Path: tools.VersionsListPath, HandlerFunc: store.GetVersionsHandler, AccessLevel: security.Admin},
	}
	security.RegisterRoutes(routes)
}
