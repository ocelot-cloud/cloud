//go:build slow

package backups

import (
	"github.com/ocelot-cloud/shared/assert"
	"ocelot/backend/apps/cloud"
	"ocelot/backend/clients"
	"ocelot/backend/tools"
	"testing"
)

func TestPruneApp(t *testing.T) {
	setup()
	defer cleanup()
	appId := cloud.SetUpAndStartSampleApp(t)
	assertNumberOfBackups(t, tools.SampleAppBackupListRequestLocal, 0)
	assert.Nil(t, clients.BackupManager.CreateBackup(appId, tools.SampleBackupDescription))
	assertNumberOfBackups(t, tools.SampleAppBackupListRequestLocal, 1)

	err := clients.BackupManager.PruneApp(appId)
	assert.Nil(t, err)
	assertNumberOfBackups(t, tools.SampleAppBackupListRequestLocal, 1)

	cloud.ExpectDockerObject(t, cloud.Network, false)
	cloud.ExpectDockerObject(t, cloud.Volume, false)
	cloud.ExpectDockerObject(t, cloud.Container, false)
}

func assertNumberOfBackups(t *testing.T, sampleAppRequest tools.BackupListRequest, expectedNumberOfBackups int) {
	backupInfos, err := clients.BackupManager.ListBackupsOfApp(sampleAppRequest)
	assert.Nil(t, err)
	assert.Equal(t, expectedNumberOfBackups, len(backupInfos))
}
