package backups

import (
	"fmt"
	"github.com/ocelot-cloud/shared/validation"
	"net/http"
	"ocelot/backend/apps/cloud"
	"ocelot/backend/apps/common"
	"ocelot/backend/clients"
	"ocelot/backend/tools"
	"os/exec"
	"strconv"
)

func AppPruneHandler(w http.ResponseWriter, r *http.Request) {
	appIdString, err := validation.ReadBody[tools.NumberString](w, r)
	if err != nil {
		return
	}

	appId, err := strconv.Atoi(appIdString.Value)
	if err != nil {
		Logger.Info("Failed to convert app id")
		http.Error(w, "Failed to convert app id", http.StatusBadRequest)
		return
	}

	err = tools.TryLockAndRespondForError(w, "prune app")
	if err != nil {
		return
	}
	defer tools.AppOperationMutex.Unlock()

	if cloud.IsOcelotDbApp(w, appId) {
		return
	}

	err = clients.BackupManager.PruneApp(appId)
	if err != nil {
		Logger.Info("Failed to prune app: %v", err)
		http.Error(w, "Failed to prune app", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (b *RealBackupManager) PruneApp(appId int) error {
	app, err := common.AppRepo.GetApp(appId)
	if err != nil {
		return err
	}

	if common.IsOcelotDbApp(*app) {
		return fmt.Errorf("cannot prune the ocelotdb app")
	}

	err = clients.Apps.StopApp(appId)
	if err != nil {
		return err
	}

	networkName := app.Maintainer + "_" + app.AppName
	networkDisConnectionCommand := fmt.Sprintf("docker network disconnect %s ocelotcloud || true", networkName)
	err = exec.Command("/bin/sh", "-c", networkDisConnectionCommand).Run() // #nosec G204 (CWE-78): Execution as root with variables in subprocess is required by design
	if err != nil {
		return err
	}

	networkDeletionCommand := fmt.Sprintf("docker network rm %s || true", networkName)
	err = exec.Command("/bin/sh", "-c", networkDeletionCommand).Run() // #nosec G204 (CWE-78): Execution as root with variables in subprocess is required by design
	if err != nil {
		return err
	}

	prefix := app.Maintainer + "_" + app.AppName + "_"
	volumeDeletionCommand := fmt.Sprintf("docker volume ls --filter name=%s --format '{{.Name}}' | grep '^%s' | xargs -r docker volume rm -f || true", prefix, prefix)
	err = exec.Command("/bin/sh", "-c", volumeDeletionCommand).Run() // #nosec G204 (CWE-78): Execution as root with variables in subprocess is required by design
	if err != nil {
		return err
	}

	err = common.AppRepo.DeleteApp(appId)
	if err != nil {
		return err
	}

	return nil
}
