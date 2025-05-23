//go:build slow

package ssh

import (
	"github.com/ocelot-cloud/shared/assert"
	"ocelot/backend/apps/common"
	"ocelot/backend/tools"
	"reflect"
	"testing"
	"time"
)

func TestKnownHostsRetrieval(t *testing.T) {
	SshClient = &SshClientReal{}
	StartSshTestContainer()
	defer ShutDownSshTestContainer()
	time.Sleep(1 * time.Second)

	remoteRepo := GetSampleRemoteRepo()
	assert.NotNil(t, SshClient.TestWhetherSshAccessWorks(remoteRepo))
	knownHosts, err := SshClient.GetKnownHosts(remoteRepo.Host, remoteRepo.SshPort)
	assert.Nil(t, err)
	remoteRepo.SshKnownHosts = knownHosts
	assert.Nil(t, SshClient.TestWhetherSshAccessWorks(remoteRepo))
}

func TestRemoteBackupRepoManagement(t *testing.T) {
	SshClient = &SshClientReal{}
	common.InitializeDatabase(false, false)
	defer common.WipeWholeDatabase()
	sampleRemoteRepo := GetSampleRemoteRepo()

	enabled, err := IsRemoteBackupEnabled()
	assert.Nil(t, err)
	assert.False(t, enabled)
	emptyRepo, err := GetRemoteBackupRepository()
	assert.Nil(t, err)
	assert.True(t, reflect.DeepEqual(emptyRepo, getEmptyRemoteRepoConfigs()))
	enabled, err = IsRemoteBackupEnabled()
	assert.Nil(t, err)
	assert.False(t, enabled)

	assert.Nil(t, SetRemoteBackupRepository(sampleRemoteRepo))
	enabled, err = IsRemoteBackupEnabled()
	assert.Nil(t, err)
	assert.True(t, enabled)

	resultRepo, err := GetRemoteBackupRepository()
	assert.Nil(t, err)
	assert.True(t, reflect.DeepEqual(sampleRemoteRepo, *resultRepo))

	resultRepo.IsEnabled = false
	assert.Nil(t, SetRemoteBackupRepository(*resultRepo))
	enabled, err = IsRemoteBackupEnabled()
	assert.Nil(t, err)
	assert.False(t, enabled)

	resultRepo, err = GetRemoteBackupRepository()
	assert.Nil(t, err)
	sampleRemoteRepo.IsEnabled = false
	assert.True(t, reflect.DeepEqual(sampleRemoteRepo, *resultRepo))
}

func TestSshClientMock_GetKnownHosts(t *testing.T) {
	SshClient = &SshClientMock{}
	_, err := SshClient.GetKnownHosts("asd", "123")
	assert.NotNil(t, err)
	assert.Equal(t, "ssh client mock says incorrect host or port", err.Error())

	knownHost, err := SshClient.GetKnownHosts("localhost", "2222")
	assert.Nil(t, err)
	assert.Equal(t, sampleKnownHost, knownHost)
}

func TestSshClientMock_TestWhetherSshAccessWorks(t *testing.T) {
	SshClient = &SshClientMock{}
	err := SshClient.TestWhetherSshAccessWorks(tools.RemoteBackupRepository{})
	assert.NotNil(t, err)
	assert.Equal(t, "ssh client mock says incorrect repo info", err.Error())

	err = SshClient.TestWhetherSshAccessWorks(tools.RemoteBackupRepository{
		IsEnabled:          false,
		Host:               "localhost",
		SshPort:            "2222",
		SshUser:            "sshadmin",
		SshPassword:        "ssh-password",
		SshKnownHosts:      sampleKnownHost,
		EncryptionPassword: "some-password",
	})
	assert.Nil(t, err)
}
