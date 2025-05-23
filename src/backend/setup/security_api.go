package setup

import (
	"errors"
	"fmt"
	"net/http"
	"ocelot/backend/clients"
	"ocelot/backend/security"
	"ocelot/backend/settings"
	"ocelot/backend/tools"
	"strings"
)

var (
	Logger = tools.Logger
)

func ApplyAuthMiddleware(w http.ResponseWriter, r *http.Request) {
	databaseHost := getHostFromDatabaseOrEmptyString()
	if !tools.Config.AreCrossOriginRequestsAllowed {
		originHeaderValue := r.Header.Get("Origin")
		originHost, err := getHostFromOriginHeaderValue(originHeaderValue)
		if err != nil {
			Logger.Info(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = security.DoesRequestComplyWithOriginPolicy(r.Host, originHost, databaseHost)
		if err != nil {
			Logger.Info("Request does not comply with origin policy: %s", err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		Logger.Debug("Request host: %s", r.Host)
		Logger.Debug("Origin host: %s", originHost)
		Logger.Debug("Database host: %s", databaseHost)
	}

	Logger.Debug("Request path: %s", r.URL.Path)
	if security.IsRequestAddressedToAnApp(r.Host, databaseHost) {
		Logger.Debug("request is proxies to app container")
		clients.Apps.ProxyRequestToTheAppsDockerContainer(w, r)
	} else {
		Logger.Debug("request to ocelot-cloud API is called")
		tools.Router.ServeHTTP(w, r)
	}
}

func getHostFromDatabaseOrEmptyString() string {
	value, err := settings.ConfigsRepo.GetValue(settings.CONFIG_HOST)
	if err != nil {
		return ""
	}
	return value
}

func getHostFromOriginHeaderValue(originHeaderValue string) (string, error) {
	var originHost string
	// Browsers set the origin header to "null" in some casual cases, so it must be allowed
	if originHeaderValue == "" || originHeaderValue == "null" {
		return "", nil
	} else {
		originParts := strings.Split(originHeaderValue, "://")
		if len(originParts) < 2 {
			errorMessage := fmt.Sprintf("invalid origin header: %s", originHeaderValue)
			return "", errors.New(errorMessage)
		} else {
			originHost = originParts[1]
		}
		return originHost, nil
	}
}
