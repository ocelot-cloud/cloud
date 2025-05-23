package common

import (
	"database/sql"
	"fmt"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/ocelot-cloud/shared/utils"
	"ocelot/backend/tools"
	"time"
)

var (
	ocelotdbMaintainerAndAppName = tools.OcelotDbMaintainer + "_" + tools.OcelotDbAppName
	ocelotdbNetworkName          = ocelotdbMaintainerAndAppName
	ocelotdbContainerName        = ocelotdbMaintainerAndAppName + "_ocelotdb"

	PostgresVersion            = "17.2"
	SamplePostgresCreationDate = time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

	DB *sql.DB
)

func IsOcelotDb(maintainer, appName string) bool {
	return maintainer == tools.OcelotDbMaintainer && appName == tools.OcelotDbAppName
}

func IsOcelotDbApp(app tools.RepoApp) bool {
	return IsOcelotDb(app.Maintainer, app.AppName)
}

func IsOcelotDbDto(app tools.AppDto) bool {
	return IsOcelotDb(app.Maintainer, app.AppName)
}

func InitializeDatabase(isDbAvailableViaDockerNetwork, useProdDbContainer bool) {
	initializeDatabase(isDbAvailableViaDockerNetwork, useProdDbContainer)
	postgresDir := getOcelotDatabaseDir(useProdDbContainer)
	if AppRepo.DoesAppExist(tools.OcelotDbMaintainer, tools.OcelotDbAppName) {
		Logger.Info(tools.OcelotDbAppName + " app already exists")
	} else {
		Logger.Info("Creating postgres app")
		appBytes, err := utils.ZipDirectoryToBytes(postgresDir)
		if err != nil {
			Logger.Error("Failed to read postgres app file: %v", err)
		}
		app := tools.RepoApp{
			AppId:                    -1,
			Maintainer:               tools.OcelotDbMaintainer,
			AppName:                  tools.OcelotDbAppName,
			VersionName:              PostgresVersion,
			VersionCreationTimestamp: SamplePostgresCreationDate,
			VersionContent:           appBytes,
			ShouldBeRunning:          true,
		}
		err = AppRepo.CreateApp(app)
		if err != nil {
			Logger.Error("Failed to create postgres app: %v", err)
		}
	}
}

func initializeDatabase(isDbAvailableViaDockerNetwork, useProdDbContainer bool) {
	ocelotDatabaseDir := getOcelotDatabaseDir(useProdDbContainer)

	err := utils.ExecuteShellCommand("docker ps --format '{{.Names}}' | grep -w " + ocelotdbContainerName)
	if err != nil {
		tools.Logger.Info("Database container does not exist. Starting...")
		err = utils.ExecuteShellCommand("docker network create " + ocelotdbNetworkName + " || true")
		if err != nil {
			Logger.Error("Failed to create network: %v", err)
		}
		composeUpCommand := fmt.Sprintf("docker compose -p %s -f %s/docker-compose.yml up -d", ocelotdbMaintainerAndAppName, ocelotDatabaseDir)
		Logger.Debug("Starting database container with command: %s", composeUpCommand)
		err = utils.ExecuteShellCommand(composeUpCommand)
		if err == nil {
			Logger.Info("Database container started successfully")
		} else {
			Logger.Fatal("Failed to start database: %v", err)
		}
	}

	var host string
	if isDbAvailableViaDockerNetwork {
		host = ocelotdbContainerName
	} else {
		host = "localhost"
	}

	defaultPostgresPort := "5432"
	DB, err = utils.WaitForPostgresDb(host, defaultPostgresPort)
	if err != nil {
		Logger.Fatal("%v", err)
	}

	utils.RunMigrations(tools.MigrationsDir, host, defaultPostgresPort)
	Logger.Info("Database initialized")
}

func getOcelotDatabaseDir(useProdDbContainer bool) string {
	var databaseProfile string
	if useProdDbContainer {
		databaseProfile += "prod"
	} else {
		databaseProfile += "dev"
	}
	return tools.DockerDir + "/ocelotdb/" + databaseProfile
}

func WipeWholeDatabase() {
	_, err := DB.Exec("DELETE FROM configs")
	if err != nil {
		Logger.Fatal("Database wipe failed: %v", err)
	}

	_, err = DB.Exec("DELETE FROM users")
	if err != nil {
		Logger.Fatal("Database wipe failed: %v", err)
	}

	_, err = DB.Exec(`
		DELETE FROM apps 
		WHERE NOT (maintainer = $1 AND app_name = $2)
	`, tools.OcelotDbMaintainer, tools.OcelotDbAppName)
	if err != nil {
		Logger.Fatal("Database wipe failed: %v", err)
	}
}
