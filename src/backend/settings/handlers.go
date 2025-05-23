package settings

import (
	"database/sql"
	"errors"
	"github.com/ocelot-cloud/shared/utils"
	"github.com/ocelot-cloud/shared/validation"
	"net/http"
	"ocelot/backend/security"
	"ocelot/backend/tools"
)

const (
	CONFIG_HOST = "HOST"
)

func InitializeSettingsModule() {
	routes := []security.Route{
		{Path: tools.SettingsHostSavePath, HandlerFunc: SaveHostHandler, AccessLevel: security.Admin},
		{Path: tools.SettingsHostReadPath, HandlerFunc: ReadHostHandler, AccessLevel: security.User},
		{Path: tools.SettingsGenerateCertificatePath, HandlerFunc: CertificateGenerationHandler, AccessLevel: security.Admin},
	}
	security.RegisterRoutes(routes)
}

type Settings struct {
	Data []KeyAndValue `json:"data"`
}

type KeyAndValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func ReadHostHandler(w http.ResponseWriter, r *http.Request) {
	value, err := ConfigsRepo.GetValue(CONFIG_HOST)
	if errors.Is(err, sql.ErrNoRows) {
		Logger.Info("host not set, setting default empty string")
		err = ConfigsRepo.SetConfigField(CONFIG_HOST, "")
		if err != nil {
			Logger.Error("Failed to set default empty string: %v", err)
			http.Error(w, "Failed to set default empty string", http.StatusInternalServerError)
			return
		}
	} else if err != nil {
		Logger.Error("Failed to get host value from ConfigsRepo: %v", err)
		http.Error(w, "Failed to get host value from ConfigsRepo", http.StatusInternalServerError)
	} else {
		utils.SendJsonResponse(w, tools.HostString{Value: value})
	}
}

func SaveHostHandler(w http.ResponseWriter, r *http.Request) {
	hostString, err := validation.ReadBody[tools.HostString](w, r)
	if err != nil {
		return
	}

	err = ConfigsRepo.SetConfigField(CONFIG_HOST, hostString.Value)
	if err != nil {
		Logger.Error("Failed to save host %s: %v", hostString.Value, err)
		http.Error(w, "Failed to save host", http.StatusInternalServerError)
		return
	}
}

type CertificateGenerationRequest struct {
	Host  string `json:"host" validate:"host"`
	Email string `json:"email" validate:"email_or_empty"`
}

func CertificateGenerationHandler(w http.ResponseWriter, r *http.Request) {
	request, err := validation.ReadBody[CertificateGenerationRequest](w, r)
	if err != nil {
		return
	}

	txtDnsRecordToCreate, err := crateCertificateViaLetsEncryptDns01Challenge(request.Host, request.Email, tools.Config.CertificateDnsChallengeClient)
	if err != nil {
		Logger.Info("Failed to create certificate: %v", err)
		http.Error(w, "Failed to create certificate", http.StatusInternalServerError)
		return
	}
	utils.SendJsonResponse(w, txtDnsRecordToCreate)
}
