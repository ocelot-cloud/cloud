package backups

import (
	"github.com/ocelot-cloud/shared/assert"
	"ocelot/backend/apps/common"
	"ocelot/backend/tools"
	"testing"
	"time"
)

func TestSetLastMaintenanceCycleExecutionDate(t *testing.T) {
	common.InitializeDatabase(false, false)
	defer WipeDatabaseForTesting()
	assert.Nil(t, SetDefaultMaintenanceSettingsIfNotExisting())

	lastExecutionDate, err := getLastMaintenanceCycleExecutionDate()
	assert.Nil(t, err)
	assert.Equal(t, UnixEpochStartTime, *lastExecutionDate)

	someNewDate := time.Date(2025, 4, 17, 1, 0, 0, 0, time.UTC)
	SetLastMaintenanceCycleExecutionDate(someNewDate)

	lastExecutionDate, err = getLastMaintenanceCycleExecutionDate()
	assert.Nil(t, err)
	assert.Equal(t, someNewDate, *lastExecutionDate)
}

func TestFindRetentionCandidates(t *testing.T) {
	backups := []tools.BackupInfo{
		{
			BackupId:                "daily1",
			BackupCreationTimestamp: time.Date(2023, 12, 31, 12, 0, 0, 0, time.UTC),
		},
		{
			BackupId:                "daily1b",
			BackupCreationTimestamp: time.Date(2023, 12, 31, 11, 0, 0, 0, time.UTC),
		},
		{
			BackupId:                "daily2",
			BackupCreationTimestamp: time.Date(2023, 12, 30, 12, 0, 0, 0, time.UTC),
		},
		{
			BackupId:                "daily3",
			BackupCreationTimestamp: time.Date(2023, 12, 29, 12, 0, 0, 0, time.UTC),
		},
		{
			BackupId:                "daily4",
			BackupCreationTimestamp: time.Date(2023, 12, 28, 12, 0, 0, 0, time.UTC),
		},
		{
			BackupId:                "daily5",
			BackupCreationTimestamp: time.Date(2023, 12, 27, 12, 0, 0, 0, time.UTC),
		},
		{
			BackupId:                "weekly2",
			BackupCreationTimestamp: time.Date(2023, 12, 24, 12, 0, 0, 0, time.UTC),
		},
		{
			BackupId:                "weekly2b",
			BackupCreationTimestamp: time.Date(2023, 12, 24, 11, 0, 0, 0, time.UTC),
		},
		{
			BackupId:                "weekly3",
			BackupCreationTimestamp: time.Date(2023, 12, 17, 12, 0, 0, 0, time.UTC),
		},
		{
			BackupId:                "weekly4",
			BackupCreationTimestamp: time.Date(2023, 12, 10, 12, 0, 0, 0, time.UTC),
		},
		{
			BackupId:                "monthly2",
			BackupCreationTimestamp: time.Date(2023, 11, 30, 12, 0, 0, 0, time.UTC),
		},
		{
			BackupId:                "monthly2b",
			BackupCreationTimestamp: time.Date(2023, 11, 29, 12, 0, 0, 0, time.UTC),
		},
		{
			BackupId:                "monthly3",
			BackupCreationTimestamp: time.Date(2022, 10, 30, 12, 0, 0, 0, time.UTC),
		},
	}
	got := FindBackupsForDeletionAccordingToRetentionPolicy(backups, 4, 3, 2)
	want := []string{"daily1b", "daily5", "weekly2b", "weekly4", "monthly2b", "monthly3"}
	assert.Equal(t, len(want), len(got))

	var gotIDs []string
	for _, b := range got {
		gotIDs = append(gotIDs, b.BackupId)
	}
	assert.Equal(t, want, gotIDs)
}

func TestManagingMaintenanceSettings(t *testing.T) {
	common.InitializeDatabase(false, false)
	defer WipeDatabaseForTesting()

	_, err := common.DB.Exec("DELETE FROM configs")
	assert.Nil(t, err)

	assert.False(t, AreMaintenanceSettingsInitialized())
	assert.Nil(t, SetDefaultMaintenanceSettingsIfNotExisting())
	assert.True(t, AreMaintenanceSettingsInitialized())

	settings, err := GetMaintenanceSettings()
	assert.Nil(t, err)
	assert.Equal(t, true, settings.AreAutoBackupsEnabled)
	assert.Equal(t, true, settings.AreAutoUpdatesEnabled)
	assert.Equal(t, 4, settings.PreferredMaintenanceHour)

	lastExecutionDate, err := getLastMaintenanceCycleExecutionDate()
	assert.Nil(t, err)
	assert.Equal(t, UnixEpochStartTime, *lastExecutionDate)

	customMaintenanceSettings := MaintenanceSettings{
		AreAutoBackupsEnabled:    false,
		AreAutoUpdatesEnabled:    false,
		PreferredMaintenanceHour: 2,
	}
	assert.Nil(t, SetMaintenanceSettings(customMaintenanceSettings))

	assert.Nil(t, SetDefaultMaintenanceSettingsIfNotExisting())
	assert.True(t, AreMaintenanceSettingsInitialized())
	settingsFromDatabase, err := GetMaintenanceSettings()
	assert.Nil(t, err)

	assert.Equal(t, customMaintenanceSettings.AreAutoBackupsEnabled, settingsFromDatabase.AreAutoBackupsEnabled)
	assert.Equal(t, customMaintenanceSettings.AreAutoUpdatesEnabled, settingsFromDatabase.AreAutoUpdatesEnabled)
	assert.Equal(t, customMaintenanceSettings.PreferredMaintenanceHour, settingsFromDatabase.PreferredMaintenanceHour)
}

func TestPreferredMaintenanceHourValidityRange(t *testing.T) {
	common.InitializeDatabase(false, false)
	defer WipeDatabaseForTesting()
	maintenanceSettings := GetDefaultMaintenanceSettings()

	maintenanceSettings.PreferredMaintenanceHour = -1
	assert.NotNil(t, SetMaintenanceSettings(maintenanceSettings))

	maintenanceSettings.PreferredMaintenanceHour = 0
	assert.Nil(t, SetMaintenanceSettings(maintenanceSettings))

	maintenanceSettings.PreferredMaintenanceHour = 23
	assert.Nil(t, SetMaintenanceSettings(maintenanceSettings))

	maintenanceSettings.PreferredMaintenanceHour = 24
	assert.NotNil(t, SetMaintenanceSettings(maintenanceSettings))
}

func TestGetLastMaintenanceCycleExecutionDate(t *testing.T) {
	defer WipeDatabaseForTesting()
	common.InitializeDatabase(false, false)
	timestamp, err := getLastMaintenanceCycleExecutionDate()
	assert.Nil(t, err)
	assert.Equal(t, UnixEpochStartTime, *timestamp)

	sampleDate := time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC)
	SetLastMaintenanceCycleExecutionDate(sampleDate)

	timestamp, err = getLastMaintenanceCycleExecutionDate()
	assert.Nil(t, err)
	assert.Equal(t, sampleDate, *timestamp)
}
