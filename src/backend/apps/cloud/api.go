package cloud

import (
	"errors"
	"fmt"
	"github.com/ocelot-cloud/shared/utils"
	"github.com/ocelot-cloud/shared/validation"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"ocelot/backend/security"
	"ocelot/backend/settings"
	"ocelot/backend/tools"
	"strconv"
	"strings"
)

type Target struct {
	Container string
	Port      string
	URL       *url.URL
}

func (a *RealAppManager) ProxyRequestToTheAppsDockerContainer(w http.ResponseWriter, r *http.Request) {
	Logger.Debug("Proxying request")

	requestHost := getHostFromRequestHost(r.Host)
	hostFromDatabase, err := settings.ConfigsRepo.GetValue(settings.CONFIG_HOST)
	if err != nil {
		Logger.Error("Failed to get host: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	target, err := getTarget(requestHost, r.URL.RequestURI(), hostFromDatabase)
	if err != nil {
		Logger.Error("Failed to get target: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	secret, isPresent, isValid := isQuerySecretPresent(r)
	if !isValid {
		Logger.Warn("invalid secret value")
		http.Error(w, "invalid input", http.StatusBadRequest)
	}
	if isPresent {
		err = exchangeSecretAgainstAuthenticationCookieAndTellBrowserToRepeatThatRequest(w, r, secret)
		if err != nil {
			Logger.Error("Failed to handle secret: %v", err)
		}
		return
	}

	if _, err = security.GetAuthentication(w, r); err != nil {
		return
	}

	proxy := createProxyRequest(r, *target)
	proxy.ServeHTTP(w, r)
}

func getHostFromRequestHost(host string) string {
	requestHost, _, _ := net.SplitHostPort(host)
	if requestHost == "" {
		requestHost = host
	}
	return requestHost
}

func isQuerySecretPresent(r *http.Request) (string, bool, bool) {
	secret := r.URL.Query().Get(tools.OcelotQuerySecretName)
	if secret == "" {
		return "", false, true
	} else {
		err := validation.ValidateSecret(secret)
		if err != nil {
			Logger.Warn("invalid secret value")
			return "", false, false
		}
		return secret, true, true
	}
}

func getTarget(requestHost, path, hostFromDatabase string) (*Target, error) {
	var target = Target{}
	if strings.HasSuffix(requestHost, hostFromDatabase) {
		target.Container = strings.TrimSuffix(requestHost, "."+hostFromDatabase)
	} else {
		Logger.Error("requestHost %s does not end with %s, but it should have", requestHost, hostFromDatabase)
		return nil, errors.New("internal error")
	}

	appConfig, ok := GetAppConfig(target.Container)
	if !ok {
		Logger.Error("app %s not found", target.Container)
		return nil, errors.New("internal error")
	}
	target.Port = strconv.Itoa(appConfig.Port)
	Logger.Debug("AppConfig: %+v\n", appConfig)

	var err error
	target.URL, err = buildTargetURL(target.Container, target.Port, path)
	Logger.Debug("proxying to target URL: %s", target.URL)
	if err != nil {
		Logger.Error("error when parsing URL, %s", err.Error())
		return nil, fmt.Errorf("error when parsing URL, %s", err.Error())
	}
	return &target, nil
}

func buildTargetURL(targetContainer, targetPort, requestURI string) (*url.URL, error) {
	rawString := fmt.Sprintf("http://%s:%s%s", targetContainer, targetPort, requestURI)
	return url.Parse(rawString)
}

func createProxyRequest(originalRequest *http.Request, target Target) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target.URL)
	proxy.Director = func(newProxyRequest *http.Request) {
		newProxyRequest.Header.Set("X-Forwarded-Host", originalRequest.Host)
		newProxyRequest.Header.Set("X-Forwarded-Proto", "https")
		newProxyRequest.Host = originalRequest.Host
		newProxyRequest.URL.Scheme = "http"
		newProxyRequest.URL.Host = fmt.Sprintf("%s:%s", target.Container, target.Port)
		query := newProxyRequest.URL.Query()
		query.Del(tools.OcelotQuerySecretName)
		newProxyRequest.URL.RawQuery = query.Encode()
		newProxyRequest.Header.Del(tools.OcelotAuthCookieName)

		removeOcelotAuthCookie(newProxyRequest, originalRequest)

	}
	return proxy
}

func removeOcelotAuthCookie(oldRequest *http.Request, newRequest *http.Request) {
	var newCookies []string
	for _, cookie := range newRequest.Cookies() {
		if cookie.Name != tools.OcelotAuthCookieName {
			newCookies = append(newCookies, cookie.String())
		}
	}
	if len(newCookies) > 0 {
		oldRequest.Header.Set("Cookie", strings.Join(newCookies, "; "))
	} else {
		oldRequest.Header.Del("Cookie")
	}
}

func exchangeSecretAgainstAuthenticationCookieAndTellBrowserToRepeatThatRequest(w http.ResponseWriter, r *http.Request, urlSecret string) error {
	err := validation.ValidateSecret(urlSecret)
	if err != nil {
		Logger.Error("validation of secret failed: %v", err)
		return errors.New("invalid input")
	}

	cookie, err := utils.GenerateCookie()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return err
	}

	cookieValue, err := security.UserRepo.GetAssociatedCookieValueAndDeleteSecret(urlSecret)
	if err != nil {
		Logger.Error("secret does not exist: %v", err)
		http.Error(w, "secret does not exist", http.StatusBadRequest)
		return err
	}

	cookie.Name = tools.OcelotAuthCookieName
	cookie.Value = cookieValue
	/*
		   This line is important. Removing it, or setting a strict mode, would break the feature that when you click the Open button in the GUI, you are correctly redirected to the app. This is because browsers see the Open button, which opens a new tab with a different URL, as a cross-request. So they do not send cookies to the second domain in that browser tab. Reloading the tab does not help. Funnily enough, the exact same URL works when you press Enter on the URL bar. I think this is because it makes the browser forget that it is a cross-request and lets it send the cookie correctly. It also works if you copy the exact same URL into a new tab and press Enter.

			I also think that this browser protection is only triggered when you use http and not when you use https, but I am not sure.
	*/
	cookie.SameSite = http.SameSiteLaxMode
	http.SetCookie(w, cookie)

	redirectURL := *r.URL
	redirectURL.RawQuery = ""
	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
	return nil
}
