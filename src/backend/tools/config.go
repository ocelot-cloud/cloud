package tools

import (
	"github.com/gorilla/mux"
	"github.com/ocelot-cloud/shared/utils"
	"os"
	"time"
)

const (
	CookieExpirationTime  = 30 * 24 * time.Hour
	TempDir               = "/tmp"
	OcelotAuthCookieName  = "ocelot-auth"
	OcelotQuerySecretName = "ocelot-secret"
	TestCookieValue       = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	ApiPath       = "/api"
	SecretPath    = ApiPath + "/secret"
	LoginPath     = ApiPath + "/login"
	WipePath      = ApiPath + "/wipe"
	CheckAuthPath = ApiPath + "/check-auth"

	UsersPath          = ApiPath + "/users"
	UsersLogoutPath    = UsersPath + "/logout"
	UsersListPath      = UsersPath + "/list"
	UsersCreatePath    = UsersPath + "/create"
	UsersDeletePath    = UsersPath + "/delete"
	ChangePasswordPath = UsersPath + "/change-password"

	VersionsPath         = ApiPath + "/versions"
	VersionsInstallPath  = VersionsPath + "/install"
	VersionsDownloadPath = VersionsPath + "/download"
	VersionsListPath     = VersionsPath + "/list"

	AppsPath       = ApiPath + "/apps"
	AppsSearchPath = AppsPath + "/search"
	AppsListPath   = AppsPath + "/list"
	AppsPrunePath  = AppsPath + "/prune"
	AppsUpdatePath = AppsPath + "/update"
	AppsStartPath  = AppsPath + "/start"
	AppsStopPath   = AppsPath + "/stop"

	BackupsPath         = ApiPath + "/backups"
	BackupsCreatePath   = BackupsPath + "/create"
	BackupsListPath     = BackupsPath + "/list"
	BackupsRestorePath  = BackupsPath + "/restore"
	BackupsDeletePath   = BackupsPath + "/delete"
	BackupsListAppsPath = BackupsPath + "/list-apps"

	SettingsPath         = ApiPath + "/settings"
	SettingsHostPath     = SettingsPath + "/host"
	SettingsHostSavePath = SettingsHostPath + "/save"
	SettingsHostReadPath = SettingsHostPath + "/read"

	SettingsCertificatePath         = SettingsPath + "/certificate"
	SettingsCertificateUploadPath   = SettingsCertificatePath + "/upload"
	SettingsGenerateCertificatePath = SettingsCertificatePath + "/generate"

	SettingsSshPath           = SettingsPath + "/ssh"
	SettingsSshReadPath       = SettingsSshPath + "/read"
	SettingsSshSavePath       = SettingsSshPath + "/save"
	SettingsSshTestAccessPath = SettingsSshPath + "/test-access"
	SettingsSshKnownHostsPath = SettingsSshPath + "/known-hosts"

	SettingsMaintenancePath     = SettingsPath + "/maintenance"
	SettingsMaintenanceSavePath = SettingsMaintenancePath + "/save"
	SettingsMaintenanceReadPath = SettingsMaintenancePath + "/read"
)

var (
	AssetsDir     = FindDirWithIdeSupport("assets")
	DockerDir     = AssetsDir + "/docker"
	MigrationsDir = AssetsDir + "/migrations"
	SampleAppDir  = DockerDir + "/sampleapp"

	Config = getGlobalConfigBasedOnProfile(Profile)
	Router = mux.NewRouter()
	Logger = utils.ProvideLogger(os.Getenv("LOG_LEVEL"))
)

type BackendProfile int

const (
	PROD BackendProfile = iota
	NATIVE
	DOCKER_TEST
)

type CertificateDnsChallengeClientType int

const (
	PRODUCTION_LETSENCRYPT_CERTIFICATE CertificateDnsChallengeClientType = iota
	FAKE_LETSENCRYPT_CERTIFICATE
	STUB_CERTIFICATE
)

type GlobalConfig struct {
	Profile                        BackendProfile
	AreCrossOriginRequestsAllowed  bool
	IsGuiEnabled                   bool
	OpenDataWipeEndpoint           bool
	UseRealAppStoreClient          bool
	IsUsingDockerNetwork           bool
	UseProductionDatabaseContainer bool
	IsMaintenanceAgentEnabled      bool
	UseMockedSshClient             bool
	CertificateDnsChallengeClient  CertificateDnsChallengeClientType
}

var Profile = GetProfile()

func (p BackendProfile) String() string {
	if p == NATIVE {
		return "NATIVE"
	} else if p == DOCKER_TEST {
		return "DOCKER_TEST"
	} else {
		return "PROD"
	}
}

type BackendComponentMode int

func GetProfile() BackendProfile {
	profile := os.Getenv("PROFILE")
	if profile == NATIVE.String() {
		return NATIVE
	} else if profile == DOCKER_TEST.String() {
		return DOCKER_TEST
	} else {
		return PROD
	}
}

func getGlobalConfigBasedOnProfile(profile BackendProfile) *GlobalConfig {
	config := GlobalConfig{}
	config.Profile = profile

	if profile == NATIVE {
		config.IsGuiEnabled = false
		config.AreCrossOriginRequestsAllowed = true
		config.OpenDataWipeEndpoint = true
		config.UseRealAppStoreClient = false
		config.IsUsingDockerNetwork = false
		config.UseProductionDatabaseContainer = false
		config.IsMaintenanceAgentEnabled = false
		config.UseMockedSshClient = true
		config.CertificateDnsChallengeClient = STUB_CERTIFICATE
		Logger = utils.ProvideLogger("DEBUG")
	} else if profile == DOCKER_TEST {
		config.IsGuiEnabled = true
		config.AreCrossOriginRequestsAllowed = false
		config.OpenDataWipeEndpoint = true
		config.UseRealAppStoreClient = false
		config.IsUsingDockerNetwork = true
		config.UseProductionDatabaseContainer = false
		config.IsMaintenanceAgentEnabled = false
		config.UseMockedSshClient = true
		config.CertificateDnsChallengeClient = FAKE_LETSENCRYPT_CERTIFICATE
		Logger = utils.ProvideLogger("DEBUG")
	} else {
		config.IsGuiEnabled = true
		config.AreCrossOriginRequestsAllowed = false
		config.OpenDataWipeEndpoint = false
		config.UseRealAppStoreClient = true
		config.IsUsingDockerNetwork = true
		config.UseProductionDatabaseContainer = true
		config.IsMaintenanceAgentEnabled = true
		config.UseMockedSshClient = false
		config.CertificateDnsChallengeClient = PRODUCTION_LETSENCRYPT_CERTIFICATE
	}

	return &config
}

func LogGlobalVariables() {
	if Profile == PROD {
		Logger.Info("Profile is: %s", Profile.String())
	} else {
		Logger.Warn("Profile is: %s. It is intended for development, so do not use this profile in production!", Profile.String())
	}
	Logger.Info("Log level is: %s", utils.GetLogLevel())
	Logger.Debug("Is web GUI enabled? -> %v", Config.IsGuiEnabled)
	Logger.Debug("Is the CORS policy relaxed by explicitly allowing cross-origin requests by setting specific response headers? -> %v", Config.AreCrossOriginRequestsAllowed)
	if Config.AreCrossOriginRequestsAllowed {
		Logger.Warn("The CORS policy is relaxed and cross-origin requests are allowed.")
	}
	if Config.UseRealAppStoreClient {
		Logger.Debug("A real app store client is used.")
	} else {
		Logger.Warn("An app store mock client is used. No data is fetched from the real app store.")
	}
	if AreMocksUsed() {
		Logger.Warn("Mock clients are used instead of restic and docker.")
	} else {
		Logger.Info("Real client for backups and containers are used via restic and docker.")
	}

}
