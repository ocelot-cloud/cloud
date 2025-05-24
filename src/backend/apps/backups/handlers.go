package backups

import (
	"github.com/ocelot-cloud/shared/utils"
	"github.com/ocelot-cloud/shared/validation"
	"net/http"
	"ocelot/backend/apps/cloud"
	"ocelot/backend/clients"
	"ocelot/backend/tools"
	"strconv"
	"strings"
)

func CreateBackupHandler(w http.ResponseWriter, r *http.Request) {
	appIdString, err := validation.ReadBody[tools.NumberString](w, r)
	if err != nil {
		return
	}

	appId, err := strconv.Atoi(appIdString.Value)
	if err != nil {
		Logger.Error("Error converting appId to int")
		http.Error(w, "Error converting appId to int", http.StatusBadRequest)
		return
	}
	err = tools.TryLockAndRespondForError(w, "create backup")
	if err != nil {
		return
	}
	defer tools.AppOperationMutex.Unlock()

	err = clients.BackupManager.CreateBackup(appId, tools.ManualBackupDescription)
	if err != nil {
		Logger.Error("Error creating backup: %v", err)
		http.Error(w, "Error creating backup", http.StatusInternalServerError)
		return
	}
}

func ListBackupsHandler(w http.ResponseWriter, r *http.Request) {
	listRequest, err := validation.ReadBody[tools.BackupListRequest](w, r)
	if err != nil {
		return
	}

	backups, err := clients.BackupManager.ListBackupsOfApp(*listRequest)
	if err != nil {
		Logger.Error("Error listing backups")
		http.Error(w, "Error listing backups", http.StatusInternalServerError)
		return
	}
	utils.SendJsonResponse(w, backups)
}

func RestoreBackupHandler(w http.ResponseWriter, r *http.Request) {
	backupRestoreRequest, err := validation.ReadBody[tools.BackupOperationRequest](w, r)
	if err != nil {
		return
	}
	err = tools.TryLockAndRespondForError(w, "restore backup")
	if err != nil {
		return
	}
	defer tools.AppOperationMutex.Unlock()

	_, err = clients.BackupManager.RestoreBackup(*backupRestoreRequest)
	if err != nil {
		Logger.Error("Error restoring backup: %v", err)
		http.Error(w, "Error restoring backup", http.StatusInternalServerError)
		return
	}
}

func DeleteBackupHandler(w http.ResponseWriter, r *http.Request) {
	deleteBackupRequest, err := validation.ReadBody[tools.BackupOperationRequest](w, r)
	if err != nil {
		return
	}

	err = tools.TryLockAndRespondForError(w, "delete backup")
	if err != nil {
		return
	}
	defer tools.AppOperationMutex.Unlock()

	err = clients.BackupManager.DeleteBackup(deleteBackupRequest.BackupId, deleteBackupRequest.IsLocal)
	if err != nil {
		Logger.Error("Error deleting backup, %v", err)
		http.Error(w, "Error deleting backup", http.StatusInternalServerError)
		return
	}
}

func VersionUpdateHandler(w http.ResponseWriter, r *http.Request) {
	appIdString, err := validation.ReadBody[tools.NumberString](w, r)
	if err != nil {
		return
	}

	appId, err := strconv.Atoi(appIdString.Value)
	if err != nil {
		Logger.Error("Failed to convert app id to int: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = tools.TryLockAndRespondForError(w, "update app")
	if err != nil {
		return
	}
	defer tools.AppOperationMutex.Unlock()

	if cloud.IsOcelotDbApp(w, appId) {
		return
	}

	err = clients.BackupManager.UpdateAppVersion(appId)
	if err != nil {
		msg := "Failed to update app version: " + err.Error()
		println("error: ", err.Error())
		if strings.Contains(err.Error(), "can't update app") {
			Logger.Info(msg)
			http.Error(w, msg, http.StatusConflict)
		} else {
			Logger.Error(msg)
			http.Error(w, "Failed to update app version", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

func ListAppsOfBackupRepository(w http.ResponseWriter, r *http.Request) {
	readFromLocalBackupRepo, err := validation.ReadBody[tools.SingleBool](w, r)
	if err != nil {
		return
	}

	apps, err := clients.BackupManager.ListAppsInBackupRepo(readFromLocalBackupRepo.Value)
	if err != nil {
		Logger.Error("Error listing apps in backup repository: %v", err)
		http.Error(w, "Error listing apps in backup repository", http.StatusInternalServerError)
		return
	}

	utils.SendJsonResponse(w, apps)
}

func GetMaintenanceSettingsHandler(w http.ResponseWriter, r *http.Request) {
	maintenanceSettings, err := GetMaintenanceSettings()
	if err != nil {
		Logger.Error("Error getting maintenance settings: %v", err)
		http.Error(w, "Error getting maintenance settings", http.StatusInternalServerError)
		return
	}

	utils.SendJsonResponse(w, maintenanceSettings)
}

func SetMaintenanceSettingsHandler(w http.ResponseWriter, r *http.Request) {
	maintenanceSettings, err := validation.ReadBody[MaintenanceSettings](w, r)
	if err != nil {
		return
	}

	err = SetMaintenanceSettings(*maintenanceSettings)
	if err != nil {
		Logger.Error("Error setting maintenance settings: %v", err)
		http.Error(w, "Error setting maintenance settings", http.StatusInternalServerError)
		return
	}
	utils.SendJsonResponse(w, maintenanceSettings)
}
