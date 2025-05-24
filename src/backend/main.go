package main

import (
	"github.com/ocelot-cloud/shared/utils"
	"github.com/spf13/cobra"
	"net/http"
	"ocelot/backend/apps"
	"ocelot/backend/apps/backups"
	"ocelot/backend/apps/cloud"
	"ocelot/backend/certs"
	"ocelot/backend/clients"
	"ocelot/backend/security"
	"ocelot/backend/settings"
	"ocelot/backend/setup"
	"ocelot/backend/ssh"
	"ocelot/backend/tools"
	"os"
)

var Logger = tools.Logger

func main() {
	rootCmd := &cobra.Command{
		Use:   "backend",
		Short: "run backend server",
		Run: func(cmd *cobra.Command, args []string) {
			runServer()
		},
	}

	generateSelfSignedCertPemCmd := &cobra.Command{
		Use:   "generate-sample-cert",
		Short: "Generate a sample self-signed certificate",
		Run: func(cmd *cobra.Command, args []string) {
			cert, err := certs.GenerateUniversalSelfSignedCert()
			if err != nil {
				Logger.Fatal("Failed to generate self-signed certificate: %v", err)
			}
			bytes, err := certs.ConvertToFullchainPemBytes(cert)
			if err != nil {
				Logger.Fatal("Failed to convert self-signed certificate to fullchain.pem bytes: %v", err)
			}
			err = os.MkdirAll("./data", 0600)
			if err != nil {
				Logger.Error("Failed to create data directory: %v", err)
			}
			err = os.WriteFile("./data/sample-fullchain.pem", bytes, 0600)
			if err != nil {
				Logger.Fatal("Failed to write fullchain.pem: %v", err)
			}
		},
	}

	rootCmd.AddCommand(generateSelfSignedCertPemCmd)
	if err := rootCmd.Execute(); err != nil {
		Logger.Fatal("Failed to execute: %v", err)
	}
}

func runServer() {
	setup.VerifyCliToolInstallations()
	createDockerNetworks()

	utils.Logger = tools.Logger
	setup.InitializeDatabase()
	tools.LogGlobalVariables()

	setClients()
	addWipeEndpointIfTestingProfileIsEnabled()

	settings.InitializeSettingsModule()
	ssh.InitializeSshModule()
	apps.InitializeAppsModule()
	security.InitializeUserModule()
	backups.InitializeBackupsModule()
	backups.StartMaintenanceAgent()
	setup.InitializeApplication()
}

func addWipeEndpointIfTestingProfileIsEnabled() {
	if tools.Config.OpenDataWipeEndpoint {
		security.RegisterRoutes([]security.Route{
			{Path: tools.WipePath, HandlerFunc: TestWipeHandler, AccessLevel: security.Anonymous},
		})
	}
}

func createDockerNetworks() {
	cloud.CreateExternalDockerNetworkAndConnectOcelotCloud(tools.OcelotDbMaintainer, tools.ResticAppName)

	if tools.Config.IsUsingDockerNetwork {
		cloud.CreateExternalDockerNetworkAndConnectOcelotCloud(tools.OcelotDbMaintainer, tools.OcelotDbAppName)
	}
}

func setClients() {
	clients.Apps = cloud.GetAppManager()
	clients.BackupManager = backups.ProvideBackupClient()
}

func TestWipeHandler(w http.ResponseWriter, r *http.Request) {
	backups.WipeDatabaseForTesting()
}
