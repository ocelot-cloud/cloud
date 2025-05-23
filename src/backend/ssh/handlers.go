package ssh

import (
	"github.com/ocelot-cloud/shared/utils"
	"github.com/ocelot-cloud/shared/validation"
	"net/http"
	"ocelot/backend/tools"
)

func SaveSshSettingsHandler(w http.ResponseWriter, r *http.Request) {
	remoteRepo, err := validation.ReadBody[tools.RemoteBackupRepository](w, r)
	if err != nil {
		return
	}

	err = tools.TryLockAndRespondForError(w, "save remote backup repository settings")
	if err != nil {
		return
	}
	defer tools.AppOperationMutex.Unlock()

	err = SetRemoteBackupRepository(*remoteRepo)
	if err != nil {
		Logger.Error("Failed to save ssh settings: %v", err)
		http.Error(w, "Failed to save ssh settings", http.StatusInternalServerError)
		return
	}
}

func ReadSshSettingsHandler(w http.ResponseWriter, r *http.Request) {
	remoteRepo, err := GetRemoteBackupRepository()
	if err != nil {
		Logger.Error("Failed to read ssh settings: %v", err)
		http.Error(w, "Failed to read ssh settings", http.StatusInternalServerError)
		return
	}
	utils.SendJsonResponse(w, remoteRepo)
}

func GetKnownHostsHandler(w http.ResponseWriter, r *http.Request) {
	repo, err := validation.ReadBody[tools.RemoteBackupRepository](w, r)
	if err != nil {
		return
	}

	knownHost, err := SshClient.GetKnownHosts(repo.Host, repo.SshPort)
	if err != nil {
		Logger.Info("Failed to get known hosts: %v", err)
		http.Error(w, "Failed to get known hosts", http.StatusBadRequest)
		return
	}
	utils.SendJsonResponse(w, tools.KnownHostsString{Value: knownHost})
}

func TestSshAccessHandler(w http.ResponseWriter, r *http.Request) {
	repo, err := validation.ReadBody[tools.RemoteBackupRepository](w, r)
	if err != nil {
		return
	}

	err = SshClient.TestWhetherSshAccessWorks(*repo)
	if err != nil {
		Logger.Info("SSH access failed: %v", err)
		http.Error(w, "SSH access failed", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}
