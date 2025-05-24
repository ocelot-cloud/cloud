package ssh

import (
	"errors"
	"ocelot/backend/settings"
	"ocelot/backend/tools"
	"os"
	"os/exec"
)

type SshClientType interface {
	GetKnownHosts(host, port string) (string, error)
	TestWhetherSshAccessWorks(repo tools.RemoteBackupRepository) error
}

type SshClientReal struct{}
type SshClientMock struct{}

var (
	Logger    = tools.Logger
	SshClient = ProvideSshClient()
)

func ProvideSshClient() SshClientType {
	if os.Getenv("USE_REAL_SSH_CLIENT") == "true" {
		return &SshClientReal{}
	}

	if tools.Config.UseMockedSshClient {
		return &SshClientMock{}
	} else {
		return &SshClientReal{}
	}
}

var RemoteRepoFields = struct {
	REMOTE_BACKUP_ENABLED             settings.ConfigFieldKey
	REMOTE_BACKUP_HOST                settings.ConfigFieldKey
	REMOTE_BACKUP_SSH_PORT            settings.ConfigFieldKey
	REMOTE_BACKUP_SSH_USER            settings.ConfigFieldKey
	REMOTE_BACKUP_SSH_PASSWORD        settings.ConfigFieldKey
	REMOTE_BACKUP_SSH_KNOWN_HOSTS     settings.ConfigFieldKey
	REMOTE_BACKUP_ENCRYPTION_PASSWORD settings.ConfigFieldKey
}{
	REMOTE_BACKUP_ENABLED:             "REMOTE_BACKUP_ENABLED",
	REMOTE_BACKUP_HOST:                "REMOTE_BACKUP_HOST",
	REMOTE_BACKUP_SSH_PORT:            "REMOTE_BACKUP_SSH_PORT",
	REMOTE_BACKUP_SSH_USER:            "REMOTE_BACKUP_SSH_USER",
	REMOTE_BACKUP_SSH_PASSWORD:        "REMOTE_BACKUP_SSH_PASSWORD",
	REMOTE_BACKUP_SSH_KNOWN_HOSTS:     "REMOTE_BACKUP_SSH_KNOWN_HOSTS",
	REMOTE_BACKUP_ENCRYPTION_PASSWORD: "REMOTE_BACKUP_ENCRYPTION_PASSWORD",
}

func ensureRemoteBackupConfigsExist() error {
	_, err := settings.ConfigsRepo.GetValue(RemoteRepoFields.REMOTE_BACKUP_ENABLED)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return SetRemoteBackupRepository(*getEmptyRemoteRepoConfigs())
		}
		return err
	}
	return nil
}

func IsRemoteBackupEnabled() (bool, error) {
	err := ensureRemoteBackupConfigsExist()
	if err != nil {
		return false, err
	}

	value, err := settings.ConfigsRepo.GetValue(RemoteRepoFields.REMOTE_BACKUP_ENABLED)
	if err != nil {
		return false, err
	}
	return value == "true", nil
}

func SetRemoteBackupRepository(backupRepo tools.RemoteBackupRepository) error {
	var isEnabledValue string
	if backupRepo.IsEnabled {
		isEnabledValue = "true"
	} else {
		isEnabledValue = "false"
	}
	fields := map[settings.ConfigFieldKey]string{
		RemoteRepoFields.REMOTE_BACKUP_ENABLED:             isEnabledValue,
		RemoteRepoFields.REMOTE_BACKUP_HOST:                backupRepo.Host,
		RemoteRepoFields.REMOTE_BACKUP_SSH_PORT:            backupRepo.SshPort,
		RemoteRepoFields.REMOTE_BACKUP_SSH_USER:            backupRepo.SshUser,
		RemoteRepoFields.REMOTE_BACKUP_SSH_PASSWORD:        backupRepo.SshPassword,
		RemoteRepoFields.REMOTE_BACKUP_SSH_KNOWN_HOSTS:     backupRepo.SshKnownHosts,
		RemoteRepoFields.REMOTE_BACKUP_ENCRYPTION_PASSWORD: backupRepo.EncryptionPassword,
	}

	for key, value := range fields {
		if err := settings.ConfigsRepo.SetConfigField(key, value); err != nil {
			return err
		}
	}

	return nil
}

func GetRemoteBackupRepository() (*tools.RemoteBackupRepository, error) {
	err := ensureRemoteBackupConfigsExist()
	if err != nil {
		return nil, err
	}

	fieldMap := map[settings.ConfigFieldKey]*string{
		RemoteRepoFields.REMOTE_BACKUP_ENABLED:             new(string),
		RemoteRepoFields.REMOTE_BACKUP_HOST:                new(string),
		RemoteRepoFields.REMOTE_BACKUP_SSH_PORT:            new(string),
		RemoteRepoFields.REMOTE_BACKUP_SSH_USER:            new(string),
		RemoteRepoFields.REMOTE_BACKUP_SSH_PASSWORD:        new(string),
		RemoteRepoFields.REMOTE_BACKUP_SSH_KNOWN_HOSTS:     new(string),
		RemoteRepoFields.REMOTE_BACKUP_ENCRYPTION_PASSWORD: new(string),
	}

	for configFieldName, ptr := range fieldMap {
		value, err := settings.ConfigsRepo.GetValue(configFieldName)
		if err != nil {
			Logger.Error("Error while retrieving remote backup repository field '%s': %v", configFieldName, err)
			return nil, err
		}
		*ptr = value
	}

	repo := &tools.RemoteBackupRepository{
		IsEnabled:          *fieldMap[RemoteRepoFields.REMOTE_BACKUP_ENABLED] == "true",
		Host:               *fieldMap[RemoteRepoFields.REMOTE_BACKUP_HOST],
		SshPort:            *fieldMap[RemoteRepoFields.REMOTE_BACKUP_SSH_PORT],
		SshUser:            *fieldMap[RemoteRepoFields.REMOTE_BACKUP_SSH_USER],
		SshPassword:        *fieldMap[RemoteRepoFields.REMOTE_BACKUP_SSH_PASSWORD],
		SshKnownHosts:      *fieldMap[RemoteRepoFields.REMOTE_BACKUP_SSH_KNOWN_HOSTS],
		EncryptionPassword: *fieldMap[RemoteRepoFields.REMOTE_BACKUP_ENCRYPTION_PASSWORD],
	}

	if repo.IsEnabled {
		return repo, nil
	}

	return repo, nil
}

func getEmptyRemoteRepoConfigs() *tools.RemoteBackupRepository {
	return &tools.RemoteBackupRepository{
		IsEnabled:          false,
		Host:               "",
		SshPort:            "",
		SshUser:            "",
		SshPassword:        "",
		EncryptionPassword: "",
	}
}

func (s *SshClientReal) GetKnownHosts(host, port string) (string, error) {
	scanCmd := exec.Command("ssh-keyscan", "-p", port, host)
	scanOut, err := scanCmd.Output()
	if err != nil {
		return "", err
	}
	return string(scanOut), nil
}

func (s *SshClientReal) TestWhetherSshAccessWorks(repo tools.RemoteBackupRepository) error {
	tempDir, err := os.CreateTemp(tools.TempDir, "known_hosts_")
	if err != nil {
		return err
	}

	knownHostsFilePath := tempDir.Name() + "known_hosts"
	err = os.WriteFile(knownHostsFilePath, []byte(repo.SshKnownHosts), 0600)
	if err != nil {
		return err
	}

	cmd := exec.Command("sshpass", "-p", repo.SshPassword, "ssh",
		"-p", repo.SshPort,
		repo.SshUser+"@"+repo.Host,
		"-o", "UserKnownHostsFile="+knownHostsFilePath,
		"-o", "StrictHostKeyChecking=yes",
		"-o", "PreferredAuthentications=password",
		"-o", "PasswordAuthentication=yes",
		"-o", "BatchMode=no",
		"-o", "ConnectTimeout=1",
		"exit",
	) // #nosec G204 (CWE-78): Execution as root with variables in subprocess is required by design
	return cmd.Run()
}

var sampleKnownHost = `[localhost]:2222 ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBE9bAskcTjEO7QC0Q91HmxJtXdQyi1VrWSXz59f2fT9NIQht5fISq3dsGqgKvV6aY1yyTN3737eNid2d/qYQJMY=
[localhost]:2222 ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCyzGYypqltUbM5bJgLaq2Be0ZjbSFVNWF36sFrCs2fUp7teNlC+x8BhHC75Hlcu8y/JueRT1W9fz1d2VOtLauGjJqAfPW6g6IR+gUfUIez+f+vG9YmJPUA0CsQ7MwN6hzupwiwYyvf37N2nQ2Ln6pQ0Ie6ZC/oaJqUAms65GmWsYq0P6Xx0mpae1wxdSBgkFa76ylpMjFlWnWzOqiHmyk+XHZ8+tH32Cs1amwucQdueNBQIfON4wUOC2076rul3T8A88/Y5QpV8iexbpzvym+7rb5aZF9yAYDyFcLvRWRT2CBblconNNYzBQHLLc8J46JDwB7DhKKCEWWEEMEX0ww/XzqF4Mk0F4vjv+5WKRPq8nDIyVuNfRcfDQsHnFv6Y++r4psitcfvAhgzUUFkcTr2axiLIeqApEzZ7bt1S1KvWWYcReBZimn6GZnEspxiLSdfmZtxey3PNcFYTx/hbgFpP+m2FFDuu68LOIGAPmjSSM6aJ6s0oL6b3Zy4PX7rBK8=
[localhost]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDBh6V6whD9B9eKKdJcZ13rreABr8UFYLHnEhY5cwx4N
`

func (s *SshClientMock) GetKnownHosts(host, port string) (string, error) {
	if host == "localhost" && port == "2222" {
		return sampleKnownHost, nil
	} else {
		return "", errors.New("ssh client mock says incorrect host or port")
	}
}

func (s *SshClientMock) TestWhetherSshAccessWorks(repo tools.RemoteBackupRepository) error {
	if repo.Host == "localhost" &&
		repo.SshPort == "2222" &&
		repo.SshUser == "sshadmin" &&
		repo.SshPassword == "ssh-password" &&
		repo.SshKnownHosts == sampleKnownHost {
		return nil
	} else {
		return errors.New("ssh client mock says incorrect repo info")
	}
}
