package ssh

import (
	"github.com/ocelot-cloud/task-runner"
	"ocelot/backend/security"
	"ocelot/backend/tools"
	"time"
)

func InitializeSshModule() {
	routes := []security.Route{
		{Path: tools.SettingsSshSavePath, HandlerFunc: SaveSshSettingsHandler, AccessLevel: security.Admin},
		{Path: tools.SettingsSshReadPath, HandlerFunc: ReadSshSettingsHandler, AccessLevel: security.Admin},
		{Path: tools.SettingsSshTestAccessPath, HandlerFunc: TestSshAccessHandler, AccessLevel: security.Admin},
		{Path: tools.SettingsSshKnownHostsPath, HandlerFunc: GetKnownHostsHandler, AccessLevel: security.Admin},
	}
	security.RegisterRoutes(routes)
}

func GetSampleRemoteRepo() tools.RemoteBackupRepository {
	return tools.RemoteBackupRepository{
		IsEnabled:          true,
		Host:               "localhost",
		SshPort:            "2222",
		SshUser:            "sshadmin",
		SshPassword:        "ssh-password",
		SshKnownHosts:      "sample-value",
		EncryptionPassword: "restic-password",
	}
}

func ShutDownSshTestContainer() {
	tr.ExecuteInDir(tools.DockerDir, "docker compose -f docker-compose.dummy-ssh.yml down")
}

func StartSshTestContainer() {
	tr.ExecuteInDir(tools.DockerDir, "bash -c 'docker compose -f docker-compose.dummy-ssh.yml down || true'")
	tr.ExecuteInDir(tools.DockerDir, "docker compose -f docker-compose.dummy-ssh.yml up -d")
	time.Sleep(1 * time.Second)
}
