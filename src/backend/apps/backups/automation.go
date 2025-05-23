package backups

import (
	"errors"
	"fmt"
	"ocelot/backend/apps/common"
	"ocelot/backend/clients"
	"ocelot/backend/settings"
	"ocelot/backend/tools"
	"sort"
	"strconv"
	"time"
)

var (
	areMaintenanceSettingsInitializedKeyword   settings.ConfigFieldKey = "ARE_MAINTENANCE_SETTINGS_INITIALIZED"
	enableAutoBackupsKeyword                   settings.ConfigFieldKey = "ENABLE_AUTO_BACKUPS"
	enableAutoUpdatesKeyword                   settings.ConfigFieldKey = "ENABLE_AUTO_UPDATES"
	latestMaintenanceCycleExecutionDateKeyword settings.ConfigFieldKey = "LAST_MAINTENANCE_CYCLE_EXECUTION_DATE"
	preferredMaintenanceHourKeyword            settings.ConfigFieldKey = "PREFERRED_MAINTENANCE_HOUR"

	UnixEpochStartTime = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

	preferredMaintenanceHourOutOfRangeError = "preferred maintenance hour must be between 0 and 23"
)

func StartMaintenanceAgent() {
	err := SetDefaultMaintenanceSettingsIfNotExisting()
	if err != nil {
		Logger.Fatal("Error setting default maintenance settings: %v", err)
	}

	if tools.Config.IsMaintenanceAgentEnabled {
		Logger.Info("Starting maintenance agent for automatic updates and backups")
		go func() {
			for {
				maintenanceSettings, err := GetMaintenanceSettings()
				if err != nil {
					Logger.Error("Error getting maintenance settings")
					continue
				}

				lastExecutionDate, err := getLastMaintenanceCycleExecutionDate()
				if err != nil {
					Logger.Error("Error getting last maintenance cycle execution date")
					continue
				}

				isMaintenanceCycleDueValue := IsMaintenanceCycleDue(time.Now(), *lastExecutionDate, maintenanceSettings.PreferredMaintenanceHour)
				if isMaintenanceCycleDueValue {
					conductMaintenanceTasks()
				}
				time.Sleep(5 * time.Minute)
			}
		}()
	}
}

func conductMaintenanceTasks() {
	err := tools.AppOperationMutex.TryLock("maintenance cycle")
	if err != nil {
		Logger.Warn("Maintenance cycle could not be started, another process is running")
	}
	defer tools.AppOperationMutex.Unlock()

	SetLastMaintenanceCycleExecutionDate(time.Now())
	apps, err := common.AppRepo.ListApps()
	if err != nil {
		Logger.Error("Error listing apps: %v", err)
		return
	}
	for _, app := range apps {
		createBackupsAndConductUpdates(app)
	}

	err = clients.BackupManager.RunRetentionPolicy()
	if err != nil {
		Logger.Error("Error running retention policy: %v", err)
	}
}

func createBackupsAndConductUpdates(app tools.RepoApp) {
	maintenanceSettings, err := GetMaintenanceSettings()
	if err != nil {
		Logger.Error("Error getting maintenance settings: %v", err)
		return
	}

	wasPreUpdateBackupCreated := false
	if maintenanceSettings.AreAutoUpdatesEnabled && !common.IsOcelotDbApp(app) {
		err := clients.BackupManager.UpdateAppVersion(app.AppId)
		if err != nil {
			Logger.Info("app was not updated: %v", err)
		} else {
			wasPreUpdateBackupCreated = true
		}
	}

	if maintenanceSettings.AreAutoBackupsEnabled && !wasPreUpdateBackupCreated {
		err := clients.BackupManager.CreateBackup(app.AppId, tools.AutoBackupDescription)
		if err != nil {
			Logger.Info("app was not backed up: %v", err)
		}
	}
}

func SetLastMaintenanceCycleExecutionDate(timestamp time.Time) {
	err := settings.ConfigsRepo.SetConfigField(latestMaintenanceCycleExecutionDateKeyword, timestamp.Format(time.RFC3339))
	if err != nil {
		Logger.Error("Error setting last maintenance cycle execution date: %v", err)
		return
	}
}

func getLastMaintenanceCycleExecutionDate() (*time.Time, error) {
	lastMaintenanceCycleExecutionDateString, err := settings.ConfigsRepo.GetValue(latestMaintenanceCycleExecutionDateKeyword)
	if err != nil {
		return nil, err
	}
	lastMaintenanceCycleExecutionDate, err := time.Parse(time.RFC3339, lastMaintenanceCycleExecutionDateString)
	if err != nil {
		Logger.Error("error parsing last maintenance cycle execution date: %v", err)
		return nil, fmt.Errorf("last maintenance cycle execution date could not be parsed")
	}
	return &lastMaintenanceCycleExecutionDate, nil
}

type MaintenanceSettings struct {
	AreAutoBackupsEnabled    bool `json:"are_auto_backups_enabled"`
	AreAutoUpdatesEnabled    bool `json:"are_auto_updates_enabled"`
	PreferredMaintenanceHour int  `json:"preferred_maintenance_hour"`
}

func AreMaintenanceSettingsInitialized() bool {
	areMaintenanceSettingsInitialized, err := settings.ConfigsRepo.GetValue(areMaintenanceSettingsInitializedKeyword)
	if err != nil {
		return false
	}
	return areMaintenanceSettingsInitialized == "true"
}

func GetMaintenanceSettings() (*MaintenanceSettings, error) {
	enableAutoBackups, err := settings.ConfigsRepo.GetValue(enableAutoBackupsKeyword)
	if err != nil {
		return nil, err
	}

	enableAutoUpdates, err := settings.ConfigsRepo.GetValue(enableAutoUpdatesKeyword)
	if err != nil {
		return nil, err
	}

	preferredMaintenanceHourString, err := settings.ConfigsRepo.GetValue(preferredMaintenanceHourKeyword)
	if err != nil {
		return nil, err
	}
	preferredMaintenanceHour, err := strconv.Atoi(preferredMaintenanceHourString)
	if err != nil {
		Logger.Error("preferred maintenance hour '%s' could not be parsed %v", preferredMaintenanceHourString, err)
		return nil, fmt.Errorf("preferred maintenance hour could not be parsed")
	}

	maintenanceSettings := MaintenanceSettings{
		AreAutoBackupsEnabled:    enableAutoBackups == "true",
		AreAutoUpdatesEnabled:    enableAutoUpdates == "true",
		PreferredMaintenanceHour: preferredMaintenanceHour,
	}

	return &maintenanceSettings, nil
}

func SetMaintenanceSettings(maintenanceSettings MaintenanceSettings) error {
	if maintenanceSettings.PreferredMaintenanceHour < 0 || maintenanceSettings.PreferredMaintenanceHour > 23 {
		return errors.New(preferredMaintenanceHourOutOfRangeError)
	}

	err := settings.ConfigsRepo.SetConfigField(enableAutoBackupsKeyword, fmt.Sprintf("%v", maintenanceSettings.AreAutoBackupsEnabled))
	if err != nil {
		return err
	}

	err = settings.ConfigsRepo.SetConfigField(enableAutoUpdatesKeyword, fmt.Sprintf("%v", maintenanceSettings.AreAutoUpdatesEnabled))
	if err != nil {
		return err
	}

	err = settings.ConfigsRepo.SetConfigField(preferredMaintenanceHourKeyword, fmt.Sprintf("%d", maintenanceSettings.PreferredMaintenanceHour))
	if err != nil {
		return err
	}

	if !AreMaintenanceSettingsInitialized() {
		err = settings.ConfigsRepo.SetConfigField(areMaintenanceSettingsInitializedKeyword, "true")
		if err != nil {
			return err
		}
	}

	return nil
}

func IsMaintenanceCycleDue(now, lastRun time.Time, preferredHour int) bool {
	wasLastMaintenanceCycleExecutedToday := wasLastMaintenanceCycleExecutedToday(now, lastRun)
	isWithinTimeRangeOfBeingExecutedToday := isWithinTimeRangeOfBeingExecutedToday(now, preferredHour)
	return isMaintenanceCycleDue(wasLastMaintenanceCycleExecutedToday, isWithinTimeRangeOfBeingExecutedToday)
}

func isMaintenanceCycleDue(wasLastMaintenanceCycleExecutedToday, isWithinTimeRangeOfBeingExecutedToday bool) bool {
	return !wasLastMaintenanceCycleExecutedToday && isWithinTimeRangeOfBeingExecutedToday
}

func isWithinTimeRangeOfBeingExecutedToday(now time.Time, preferredHour int) bool {
	start := time.Date(now.Year(), now.Month(), now.Day(), preferredHour, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	return !now.Before(start) && now.Before(end)
}

func wasLastMaintenanceCycleExecutedToday(now, lastRun time.Time) bool {
	return now.Year() == lastRun.Year() && now.YearDay() == lastRun.YearDay()
}

func FindBackupsForDeletionAccordingToRetentionPolicy(backups []tools.BackupInfo, keepDaily, keepWeekly, keepMonthly int) []tools.BackupInfo {
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].BackupCreationTimestamp.After(backups[j].BackupCreationTimestamp)
	})
	var dailyBackups, weeklyBackups, monthlyBackups []tools.BackupInfo
	dailySet := map[string]bool{}
	weeklySet := map[string]bool{}
	monthlySet := map[string]bool{}
	for _, backup := range backups {
		dayKey := backup.BackupCreationTimestamp.Format("2006-01-02")
		year, week := backup.BackupCreationTimestamp.ISOWeek()
		weekKey := fmt.Sprintf("%04d-%02d", year, week)
		monthKey := backup.BackupCreationTimestamp.Format("2006-01")
		if len(dailyBackups) < keepDaily && !dailySet[dayKey] {
			dailyBackups = append(dailyBackups, backup)
			dailySet[dayKey] = true
		}
		if len(weeklyBackups) < keepWeekly && !weeklySet[weekKey] {
			weeklyBackups = append(weeklyBackups, backup)
			weeklySet[weekKey] = true
		}
		if len(monthlyBackups) < keepMonthly && !monthlySet[monthKey] {
			monthlyBackups = append(monthlyBackups, backup)
			monthlySet[monthKey] = true
		}
	}
	retainedBackupIds := map[string]bool{}
	for _, backup := range append(append(dailyBackups, weeklyBackups...), monthlyBackups...) {
		retainedBackupIds[backup.BackupId] = true
	}
	var candidatesForDeletion []tools.BackupInfo
	for _, backup := range backups {
		if !retainedBackupIds[backup.BackupId] {
			candidatesForDeletion = append(candidatesForDeletion, backup)
		}
	}
	return candidatesForDeletion
}

func SetDefaultMaintenanceSettingsIfNotExisting() error {
	if AreMaintenanceSettingsInitialized() {
		return nil
	} else {
		SetLastMaintenanceCycleExecutionDate(UnixEpochStartTime)
		return SetMaintenanceSettings(GetDefaultMaintenanceSettings())
	}
}

func GetDefaultMaintenanceSettings() MaintenanceSettings {
	return MaintenanceSettings{
		AreAutoBackupsEnabled:    true,
		AreAutoUpdatesEnabled:    true,
		PreferredMaintenanceHour: 4,
	}
}
