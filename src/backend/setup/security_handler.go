package setup

import (
	"github.com/ocelot-cloud/shared/utils"
	"github.com/ocelot-cloud/shared/validation"
	"net/http"
	"ocelot/backend/security"
	"ocelot/backend/tools"
	"time"
)

func loginHandler(w http.ResponseWriter, r *http.Request) {
	creds, err := validation.ReadBody[security.Credentials](w, r)
	if err != nil {
		return
	}

	if !security.UserRepo.IsPasswordCorrect(creds.Username, creds.Password) {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	cookie, err := utils.GenerateCookie()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if creds.Username == "admin" && tools.Config.Profile == tools.NATIVE {
		cookie.Value = tools.TestCookieValue
	}
	cookie.Name = tools.OcelotAuthCookieName
	if tools.Config.Profile == tools.NATIVE {
		cookie.SameSite = http.SameSiteLaxMode
	}
	http.SetCookie(w, cookie)

	err = security.UserRepo.SaveCookie(creds.Username, cookie.Value, time.Now().UTC().Add(tools.CookieExpirationTime))
	if err != nil {
		http.Error(w, "saving cookie failed", http.StatusInternalServerError)
		return
	}

	Logger.Debug("login successful")
	w.WriteHeader(http.StatusOK)
}

func SecretHandler(w http.ResponseWriter, r *http.Request) {
	Logger.Debug("SecretHandler called")
	cookie, err := r.Cookie(tools.OcelotAuthCookieName)
	if err != nil {
		Logger.Warn("cookie not found: %v", err)
		http.Error(w, "cookie not found", http.StatusUnauthorized)
		return
	}

	secret, err := security.UserRepo.GenerateSecret(cookie.Value)
	if err != nil {
		http.Error(w, "secret generation failed", http.StatusInternalServerError)
		return
	}

	utils.SendJsonResponse(w, secret)
}
