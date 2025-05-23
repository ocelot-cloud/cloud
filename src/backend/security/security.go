package security

import (
	"context"
	"errors"
	"fmt"
	"github.com/ocelot-cloud/shared/utils"
	"github.com/ocelot-cloud/shared/validation"
	"net/http"
	"ocelot/backend/tools"
	"regexp"
	"strings"
)

type contextKey string

const authKey = contextKey("auth")

type AccessLevelType int

const (
	Anonymous AccessLevelType = iota
	User
	Admin
)

type Route struct {
	Path        string
	HandlerFunc http.HandlerFunc
	AccessLevel AccessLevelType
}

func RegisterRoutes(routes []Route) {
	for _, r := range routes {
		tools.Router.Handle(r.Path, protect(r.HandlerFunc, r.AccessLevel))
	}
}

func protect(next http.HandlerFunc, level AccessLevelType) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if level == Anonymous {
			next.ServeHTTP(w, r)
			return
		}

		r, err := GetRequestWithAuthContext(w, r)
		if err != nil {
			return
		}

		auth, err := GetAuthFromContext(w, r)
		if err != nil {
			return
		}

		hasAccess := HasAccess(level, auth)
		if hasAccess {
			next.ServeHTTP(w, r)
		} else {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
	})
}

func HasAccess(level AccessLevelType, auth *tools.Authorization) bool {
	if level == Anonymous {
		return true
	} else if auth == nil {
		return false
	}

	if level == User {
		return true
	} else {
		return auth.IsAdmin
	}
}

var crossRequestsToAppsOnlyFromOcelotCloudOriginErrorMessage = "cross request is only allowed from ocelot-cloud origin"
var CrossRequestsToOcelotCloudNotAllowedErrorMessage = "cross request to ocelot-cloud is not allowed"

func DoesRequestComplyWithOriginPolicy(requestHost, originHost, serverHost string) error {
	isCrossRequest := isCrossOriginRequest(requestHost, originHost)

	if IsRequestAddressedToAnApp(requestHost, serverHost) {
		if serverHost == "" {
			Logger.Error("this block should never be triggered, as a request to e.g. sample.localhost without setting the (server) HOST, is not considered an app request")
			return errors.New("HOST variable must be set on server to allow app requests")
		} else if isCrossRequest {
			if originHost != serverHost && originHost != "ocelot-cloud."+serverHost {
				return errors.New(crossRequestsToAppsOnlyFromOcelotCloudOriginErrorMessage)
			}
		}
	} else if isCrossRequest {
		return errors.New(CrossRequestsToOcelotCloudNotAllowedErrorMessage)
	}
	return nil
}

func isCrossOriginRequest(requestHost, originHost string) bool {
	return originHost != "" && originHost != requestHost
}

func IsRequestAddressedToAnApp(requestHost, serverHost string) bool {
	Logger.Debug("checking if request is addressed to an app, config host: '%s', request host: '%s'", serverHost, requestHost)
	pattern := fmt.Sprintf(`^.*\.%s(:\d+)?$`, serverHost)
	re := regexp.MustCompile(pattern)
	isRequestAddressedToAnApp := re.MatchString(requestHost) && !strings.HasPrefix(requestHost, "ocelot-cloud.")
	Logger.Debug("is request addressed to an app: %v", isRequestAddressedToAnApp)
	return isRequestAddressedToAnApp
}

func GetAuthFromContext(w http.ResponseWriter, r *http.Request) (*tools.Authorization, error) {
	if r.Context() == nil {
		Logger.Error("request context is nil")
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("request context is nil")
	}

	val := r.Context().Value(authKey)
	if val == nil {
		Logger.Error("auth not found in context, but protected handlers must always have an auth context")
		w.WriteHeader(http.StatusInternalServerError)
		return nil, fmt.Errorf("authorization not found in context")
	}

	auth, ok := val.(tools.Authorization)
	if !ok {
		tools.Logger.Error("auth context value is of invalid type")
		w.WriteHeader(http.StatusInternalServerError)
		return nil, fmt.Errorf("invalid authorization type in context")
	}

	return &auth, nil
}

func GetAuthentication(w http.ResponseWriter, r *http.Request) (*tools.Authorization, error) {
	cookie, err := r.Cookie(tools.OcelotAuthCookieName)
	if err != nil {
		Logger.Info("cookie required, but none found: %v", err)
		http.Error(w, "cookie required, but none found", http.StatusUnauthorized)
		return nil, fmt.Errorf("")
	}

	err = validation.ValidateSecret(cookie.Value)
	if err != nil {
		Logger.Info("invalid input of cookie value: %v", err)
		http.Error(w, "invalid input", http.StatusBadRequest)
		return nil, err
	}

	isExpired, err := UserRepo.IsExpired(cookie.Value)
	if err != nil {
		Logger.Error("could not check if cookie is expired: %v", err)
		http.Error(w, "cookie not found", http.StatusBadRequest)
		return nil, fmt.Errorf("")
	}

	if isExpired {
		Logger.Debug("cookie expired")
		Logger.Info("cookie expired")
		http.Error(w, "cookie expired", http.StatusUnauthorized)
		return nil, fmt.Errorf("")
	} else {
		Logger.Debug("cookie not expired")
		err = UserRepo.UpdateCookieExpirationDate(cookie.Value)
		if err != nil {
			Logger.Error("could not update cookie expiration date: %v", err)
			http.Error(w, "cookie not found", http.StatusInternalServerError)
			return nil, fmt.Errorf("")
		}
	}

	auth, err := UserRepo.GetAuthenticationViaCookie(cookie.Value)
	if err != nil {
		http.Error(w, "cookie not found", http.StatusUnauthorized)
		return nil, fmt.Errorf("")
	}

	return auth, nil
}

func GetRequestWithAuthContext(w http.ResponseWriter, r *http.Request) (*http.Request, error) {
	auth, err := GetAuthentication(w, r)
	if err != nil {
		return nil, err
	}

	// This simply adds context information to the request object for further security processing in the backend. The context information is not used in any http request that is proxied to an application.
	ctx := context.WithValue(r.Context(), authKey, *auth)
	r = r.WithContext(ctx)

	return r, nil
}

func CheckAuthHandler(w http.ResponseWriter, r *http.Request) {
	r, err := GetRequestWithAuthContext(w, r)
	if err != nil {
		return
	}

	auth, ok := r.Context().Value(authKey).(tools.Authorization)
	if !ok {
		Logger.Error("auth not found in context, but protected handlers must always have an auth context")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	utils.SendJsonResponse(w, auth)
}
