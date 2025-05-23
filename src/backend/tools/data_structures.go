package tools

import (
	"time"
)

type RepoApp struct {
	AppId                            int
	Maintainer, AppName, VersionName string
	VersionCreationTimestamp         time.Time
	VersionContent                   []byte
	ShouldBeRunning                  bool
}

type FullAppInfo struct {
	Maintainer               string
	AppName                  string
	VersionName              string
	VersionCreationTimestamp string
	Port                     string
	Path                     string
	IsAvailable              bool
}

type AppDto struct {
	Maintainer  string `json:"maintainer"`
	AppName     string `json:"app_name"`
	VersionName string `json:"version_name"`
	AppId       string `json:"app_id"`
	UrlPath     string `json:"url_path"`
	Status      string `json:"status"`
}

type VersionInfo struct {
	Id                       string    `json:"id"`
	Name                     string    `json:"name"`
	VersionCreationTimestamp time.Time `json:"version_creation_timestamp"`
}

type AppWithLatestVersion struct {
	Maintainer        string `json:"maintainer"`
	AppId             string `json:"app_id"`
	AppName           string `json:"app_name"`
	LatestVersionId   string `json:"latest_version_id"`
	LatestVersionName string `json:"latest_version_name"`
}

type VersionMetaData struct {
	Name              string
	CreationTimestamp time.Time
	Content           []byte
}

type FullVersionInfo struct {
	Maintainer               string    `json:"maintainer"`
	AppName                  string    `json:"app_name"`
	VersionName              string    `json:"version_name"`
	Content                  []byte    `json:"content"`
	VersionCreationTimestamp time.Time `json:"version_creation_timestamp"`
}

type Authorization struct {
	UserId               int       `json:"user_id"`
	User                 string    `json:"user"`
	IsAdmin              bool      `json:"is_admin"`
	CookieExpirationDate time.Time `json:"cookie_expiration_date,omitempty"`
	DidAcceptEula        bool      `json:"did_accept_eula"`
}

type RemoteBackupRepository struct {
	IsEnabled          bool   `json:"is_enabled"`
	Host               string `json:"host" validate:"remote_host"`
	SshPort            string `json:"ssh_port" validate:"number"`
	SshUser            string `json:"ssh_user" validate:"user_name"`
	SshPassword        string `json:"ssh_password" validate:"password"`
	SshKnownHosts      string `json:"ssh_known_hosts" validate:"known_hosts"`
	EncryptionPassword string `json:"encryption_password" validate:"password"`
}

type RestoredVersionInfo struct {
	Maintainer     string
	AppName        string
	VersionName    string
	VersionContent []byte
}

type BackupDescription string

var (
	AutoBackupDescription   BackupDescription = "auto-backup"
	ManualBackupDescription BackupDescription = "manual-backup"
)

type BackupInfo struct {
	BackupId                 string            `json:"backup_id"`
	Maintainer               string            `json:"maintainer"`
	AppName                  string            `json:"app_name"`
	VersionName              string            `json:"version_name"`
	VersionCreationTimestamp time.Time         `json:"version_creation_timestamp"`
	Description              BackupDescription `json:"description"`
	BackupCreationTimestamp  time.Time         `json:"backup_creation_timestamp"`
}

type UserFullInfo struct {
	Id                   int
	UserName             string
	HashedPassword       string
	HashedCookieValue    *string
	CookieExpirationDate *string
	IsAdmin              bool
}

type AppSearchRequest struct {
	SearchTerm         string `json:"search_term" validate:"search_term"`
	ShowUnofficialApps bool   `json:"show_unofficial_apps"`
}

type MaintainerAndApp struct {
	Maintainer string `json:"maintainer"`
	AppName    string `json:"app_name"`
}

type BackupListRequest struct {
	Maintainer string `json:"maintainer" validate:"user_name"`
	AppName    string `json:"app_name" validate:"app_name"`
	IsLocal    bool   `json:"is_local"`
}

var (
	SampleAppBackupListRequestLocal = BackupListRequest{
		Maintainer: SampleMaintainer,
		AppName:    SampleApp,
		IsLocal:    true,
	}
	SampleAppBackupListRequestRemote = BackupListRequest{
		Maintainer: SampleMaintainer,
		AppName:    SampleApp,
		IsLocal:    false,
	}
	OcelotDbAppBackupListRequestLocal = BackupListRequest{
		Maintainer: OcelotDbMaintainer,
		AppName:    OcelotDbAppName,
		IsLocal:    true,
	}
	OcelotDbAppBackupListRequestRemote = BackupListRequest{
		Maintainer: OcelotDbMaintainer,
		AppName:    OcelotDbAppName,
		IsLocal:    false,
	}
)

type SingleBool struct {
	Value bool `json:"value"`
}

type BackupOperationRequest struct {
	BackupId string `json:"backup_id" validate:"restic_backup_id"`
	IsLocal  bool   `json:"is_local"`
}

type NumberString struct {
	Value string `json:"value" validate:"number"`
}

type PasswordString struct {
	Value string `json:"value" validate:"password"`
}

type HostString struct {
	Value string `json:"value" validate:"host"`
}

type KnownHostsString struct {
	Value string `json:"value" validate:"known_hosts"`
}

type UserNameString struct {
	Value string `json:"value" validate:"user_name"`
}
