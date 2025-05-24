<template>
  <FrameComponent>
    <PageHeader title="Backups" doc-path="/ocelot-cloud/backups"></PageHeader>
    <br>
    <div  class="d-flex justify-center gap-4" style="gap: 20px;">
      <v-select
          id="backup-repository-selector"
          v-model="isLocal"
          :items="[
    { label: 'Local', value: true },
    ...(isRemoteBackupEnabled ? [{ label: 'Remote', value: false }] : [])
  ]"
          item-title="label"
          label="Please select backup repository type"
          style="max-width: 300px;"
      />
      <div id="app-selector">
        <v-select
          v-model="selectedAppId"
          :items="apps?.map(a => ({ label: a.maintainer + ' / ' + a.app_name, value: a.app_id })) || []"
          item-title="label"
          label="Please select an app from that repository"
          style="width: 300px;"
          @update:modelValue="onAppSelected"
        />
      </div>
    </div>
    <v-data-table :headers="headers" :items="backups">
      <template #item.actions="{ item }">
        <td>
          <v-menu>
            <template #activator="{ props }">
              <v-btn id="actions-button" color="primary" icon v-bind="props">
                <v-icon>mdi-cog</v-icon>
              </v-btn>
            </template>
            <v-list>
              <v-list-item id="restore-action" @click="confirmRestoreBackup(item.backup_id)">
                <v-icon left>mdi-backup-restore</v-icon> Restore
              </v-list-item>
              <v-list-item id="delete-action" @click="confirmDeleteBackup(item.backup_id)">
                <v-icon left>mdi-delete</v-icon> Delete
              </v-list-item>
            </v-list>
          </v-menu>
        </td>
      </template>
    </v-data-table>
    <ConfirmationDialog
        v-model:visible="showRestoreConfirmation"
        :on-confirm="restoreBackup"
        title="Backup Restore Confirmation"
        message="Restoring this backup will overwrite existing app data if the app is currently installed. We recommend that you create a backup first to avoid any potential data loss. Are you sure you want to restore this backup?"
    />
    <ConfirmationDialog
        v-model:visible="showDeleteConfirmation"
        :on-confirm="deleteBackup"
        title="Backup Deletion Confirmation"
        message="Are you sure you want to delete this backup?"
    />
  </FrameComponent>
</template>

<script lang="ts">
import {defineComponent, ref, onMounted, watch} from 'vue'
import FrameComponent from "@/components/FrameComponent.vue"
import { doCloudRequest } from "@/components/requests"
import PageHeader from "@/components/PageHeader.vue";
import {backendReadSshSettingsPath} from "@/components/config";
import ConfirmationDialog from "@/components/ConfirmationDialog.vue";

interface BackupInfo {
  backup_id: string
  maintainer: string
  app_name: string
  version_name: string
  version_creation_timestamp: string
  description: string
  backup_creation_timestamp: string
}

interface MaintainerAndAppName {
  app_id: string
  maintainer: string
  app_name: string
}

export default defineComponent({
  name: 'BackupsComponent',
  components: {ConfirmationDialog, PageHeader, FrameComponent },
  setup() {
    const backups = ref<BackupInfo[]>([])
    const apps = ref<MaintainerAndAppName[]>([])
    const isLocal = ref<boolean>(true)
    const selectedAppId = ref<string>('')
    const isRemoteBackupEnabled = ref(false)

    const idOfBackupToDelete = ref("")
    const showDeleteConfirmation = ref(false)

    const idOfBackupToRestore = ref("")
    const showRestoreConfirmation = ref(false)

    const headers = [
      { title: 'Backup Creation Date', key: 'backup_creation_timestamp'},
      { title: 'Version Name', key: 'version_name'},
      { title: 'Version Creation Date', key: 'version_creation_timestamp'},
      { title: 'Description', key: 'description'},
      { title: 'Actions', key: 'actions', sortable: false },
    ]

    const fetchBackups = async (maintainer: string, app_name: string, is_local: boolean) => {
      await doCloudRequest('/api/backups/list', { maintainer, app_name, is_local })
          .then(response => {
            console.log(maintainer, app_name);
            if (response && response.status === 200) {
              backups.value = Array.isArray(response.data)
                  ? response.data
                      .map(backup => ({
                        ...backup,
                        version_creation_timestamp: formatTimestamp(backup.version_creation_timestamp),
                        backup_creation_timestamp: formatTimestamp(backup.backup_creation_timestamp),
                      }))
                      .sort((a, b) => new Date(b.backup_creation_timestamp).getTime() - new Date(a.backup_creation_timestamp).getTime()) // Sort by date
                  : [];
            }
          });
    };

    const fetchApps = async () => {
      const response = await doCloudRequest("/api/backups/list-apps", {value: isLocal.value})
      if (response && response.status === 200) {
        let appList = response.data as MaintainerAndAppName[]
        if (appList && appList.length > 0) {
          apps.value = appList.map((app, index) => ({
            maintainer: app.maintainer,
            app_name: app.app_name,
            app_id: index.toString()
          }))
        } else {
          apps.value = []
        }
      }
    }

    const fetchBackupsOfSelectedApp = () => {
      const chosen = apps.value.find(a => a.app_id === selectedAppId.value)
      if (chosen) fetchBackups(chosen.maintainer, chosen.app_name, isLocal.value)
    }

    onMounted(async () => {
      await fetchApps()
      isRemoteBackupEnabled.value = await isRemoteBackupRepositoryEnabled()
      console.log("isRemoteBackupEnabled: ", isRemoteBackupEnabled.value)
    })

    const deleteBackup = async () => {
      showDeleteConfirmation.value = false
      console.log(apps.value)
      let response = await doCloudRequest("/api/backups/delete", { backup_id: idOfBackupToDelete.value, is_local: isLocal.value })
      if (response && response.status === 200) {
        if (apps.value.length == 1) {
          apps.value = []
          backups.value = []
          selectedAppId.value = ''
        } else {
          const chosen = apps.value.find(a => a.app_id === selectedAppId.value)
          if (chosen) await fetchBackups(chosen.maintainer, chosen.app_name, isLocal.value)
        }
      }
    }

    const confirmDeleteBackup = (backupId: string) => {
      idOfBackupToDelete.value = backupId
      showDeleteConfirmation.value = true
    }

    const restoreBackup = async () => {
      showRestoreConfirmation.value = false
      let response = await doCloudRequest("/api/backups/restore", { backup_id: idOfBackupToRestore.value, is_local: isLocal.value })
      if (response && response.status === 200) {
        alert("Backup restored successfully")
        const chosen = apps.value.find(a => a.app_id === selectedAppId.value)
        if (chosen) await fetchBackups(chosen.maintainer, chosen.app_name, isLocal.value)
      }
    }

    const confirmRestoreBackup = (backupId: string) => {
      idOfBackupToRestore.value = backupId
      showRestoreConfirmation.value = true
    }

    const isRemoteBackupRepositoryEnabled = async (): Promise<boolean> => {
      const resp = await doCloudRequest(backendReadSshSettingsPath, null)
      if (resp && resp.status === 200) {
        return resp.data.is_enabled
      }
      return false
    }

    watch(isLocal, () => {
      apps.value = []
      backups.value = []
      selectedAppId.value = ""
      fetchApps()
     })

    const formatTimestamp = (timestamp: string) => {
      const date = new Date(timestamp);
      const year = date.getFullYear();
      const month = String(date.getMonth() + 1).padStart(2, '0');
      const day = String(date.getDate()).padStart(2, '0');
      const hours = String(date.getHours()).padStart(2, '0');
      const minutes = String(date.getMinutes()).padStart(2, '0');
      const seconds = String(date.getSeconds()).padStart(2, '0');
      return `${year}/${month}/${day} ${hours}:${minutes}:${seconds}`;
    };

    return {
      backups,
      headers,
      deleteBackup,
      confirmDeleteBackup,
      showDeleteConfirmation,
      restoreBackup,
      confirmRestoreBackup,
      showRestoreConfirmation,
      apps,
      selectedAppId,
      onAppSelected: fetchBackupsOfSelectedApp,
      isLocal,
      isRemoteBackupEnabled,
    }
  },
})
</script>
