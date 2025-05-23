package common

import (
	"errors"
	"fmt"
	"github.com/ocelot-cloud/shared/utils"
	"ocelot/backend/tools"
	"time"
)

var AppRepo = AppRepository{}

type AppRepository struct{}

func (n AppRepository) CreateApp(app tools.RepoApp) error {
	var timestamp = app.VersionCreationTimestamp.Format(time.RFC3339)
	if n.DoesAppExist(app.Maintainer, app.AppName) {
		return fmt.Errorf("app already exists")
	}
	if _, err := DB.Exec("INSERT INTO apps (maintainer, app_name, version_name, version_creation_timestamp, version_content, should_be_running) VALUES ($1, $2, $3, $4, $5, $6)",
		app.Maintainer, app.AppName, app.VersionName, timestamp, app.VersionContent, app.ShouldBeRunning); err != nil {
		Logger.Error("failed to create app: %v", err)
		return fmt.Errorf("failed to create app")
	}
	return nil
}

func (n AppRepository) GetAppId(maintainer, app string) (int, error) {
	var appId int
	if err := DB.QueryRow("SELECT app_id FROM apps WHERE maintainer = $1 AND app_name = $2", maintainer, app).Scan(&appId); err != nil {
		Logger.Error("failed to get app id: %v", err)
		return 0, fmt.Errorf("failed to get app id")
	}
	return appId, nil
}

func (n AppRepository) GetApp(appId int) (*tools.RepoApp, error) {
	var app tools.RepoApp
	var versionCreationTimestamp string
	if err := DB.QueryRow("SELECT app_id, maintainer, app_name, version_name, version_creation_timestamp, version_content, should_be_running FROM apps WHERE app_id = $1", appId).Scan(&app.AppId, &app.Maintainer, &app.AppName, &app.VersionName, &versionCreationTimestamp, &app.VersionContent, &app.ShouldBeRunning); err != nil {
		Logger.Error("failed to get app: %v", err)
		return nil, fmt.Errorf("failed to get app")
	}
	var err error
	app.VersionCreationTimestamp, err = time.Parse(time.RFC3339, versionCreationTimestamp)
	if err != nil {
		Logger.Error("failed to parse version creation timestamp", err)
		return nil, errors.New("failed to parse version creation timestamp")
	}
	return &app, nil
}

func (n AppRepository) ListApps() ([]tools.RepoApp, error) {
	var apps []tools.RepoApp
	rows, err := DB.Query("SELECT app_id, maintainer, app_name, version_name, version_creation_timestamp, version_content, should_be_running FROM apps")
	if err != nil {
		Logger.Error("failed to list apps: %v", err)
		return nil, fmt.Errorf("failed to list apps")
	}
	defer utils.Close(rows)

	for rows.Next() {
		var app tools.RepoApp
		var versionCreationTimestamp string
		if err := rows.Scan(&app.AppId, &app.Maintainer, &app.AppName, &app.VersionName, &versionCreationTimestamp, &app.VersionContent, &app.ShouldBeRunning); err != nil {
			Logger.Error("failed to scan app: %v", err)
			return nil, fmt.Errorf("failed to scan app")
		}
		app.VersionCreationTimestamp, err = time.Parse(time.RFC3339, versionCreationTimestamp)
		if err != nil {
			Logger.Error("failed to parse version creation timestamp", err)
			return nil, errors.New("failed to parse version creation timestamp")
		}
		apps = append(apps, app)
	}

	if err := rows.Err(); err != nil {
		Logger.Error("rows error: %v", err)
		return nil, fmt.Errorf("rows error")
	}

	return apps, nil
}

func (n AppRepository) DeleteApp(appId int) error {
	if _, err := DB.Exec("DELETE FROM apps WHERE app_id = $1", appId); err != nil {
		Logger.Error("failed to delete app: %v", err)
		return fmt.Errorf("failed to delete app")
	}
	return nil
}

func (n AppRepository) SetAppShouldBeRunning(appId int, shouldBeRunning bool) error {
	if _, err := DB.Exec("UPDATE apps SET should_be_running = $1 WHERE app_id = $2", shouldBeRunning, appId); err != nil {
		Logger.Error("failed to update should_be_running: %v", err)
		return errors.New("failed to set app should be running")
	}
	return nil
}

func (n AppRepository) UpdateVersion(appId int, version tools.VersionMetaData) error {
	var timestamp = version.CreationTimestamp.Format(time.RFC3339)
	if _, err := DB.Exec("UPDATE apps SET version_name = $1, version_creation_timestamp = $2, version_content = $3 WHERE app_id = $4",
		version.Name, timestamp, version.Content, appId); err != nil {
		Logger.Error("failed to update version: %v", err)
		return fmt.Errorf("failed to update version")
	}
	return nil
}

func (n AppRepository) DoesAppExist(maintainer string, name string) bool {
	var exists bool
	if err := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM apps WHERE maintainer = $1 AND app_name = $2)", maintainer, name).Scan(&exists); err != nil {
		return false
	}
	return exists
}
