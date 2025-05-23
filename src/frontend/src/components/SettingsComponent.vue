<template>
  <FrameComponent v-if="!isDemoDomain">
    <PageHeader title="Settings" doc-path="/ocelot-cloud/settings"/>
    <br>
    <v-card class="pa-6">
      <h2>
        <span>Host</span>
        <DocsReference doc-path="/ocelot-cloud/settings/host"/>
      </h2>
      <ValidationInput
          class="mt-2"
          style="width: 300px"
          id="host-text-field"
          validationType="host"
          v-model="host"
          :submitted="wasHostSubmitted"
          label="enter host value"
      />
      <v-btn id="save-button" color="primary" @click="saveHost">Save</v-btn>
    </v-card>
    <br>
    <v-card class="pa-6">
      <v-form>
        <h2>
          <span>Certificate</span>
          <DocsReference doc-path="/ocelot-cloud/settings/certificate"/>
        </h2>
        <h3>Option 1 - Upload</h3>
        <v-btn class="mt-2 mb-2" color="primary" @click="certificateFileInput?.click()">Upload .pem file</v-btn>
        <input ref="certificateFileInput" type="file" class="d-none" @change="handleFileUpload" />
        <br>
        <br>
        <h3>Option 2 - Generation</h3>
        <p>If you own a domain, you can create a wildcard certificate by completing the Let's Encrypt DNS-01 challenge with the above host value and email address listed above.</p>
        <ValidationInput
            class="mt-2"
            style="width: 300px"
            id="certificate-generation-email-field"
            validationType="email_or_empty"
            v-model="certGenerationEmail"
            :submitted="wasCertificateGenerationRequestSubmitted"
            label="email - optional, can be left empty"
        />
        <v-btn
            id="start-dns-01-challenge-button"
            class="mb-2"
            color="primary"
            @click="startDnsChallenge"
        >
          Start DNS-01 Challenge
        </v-btn>
        <div v-if="wasCertificateGenerationRequestSuccessful">
          <p>Using your DNS provider, create a TXT record with name '{{ dns01ChallengeRecordName }}' and the value below. Wait a few minutes for the DNS record to propagate. Ocelot-Cloud will automatically create and load the new certificate.</p>
          <v-card class="mt-2 pa-2">
            <v-row style="width: 550px" no-gutters align="center">
              <v-col>
                <v-textarea
                    id="dns-01-challenge-record-value"
                    :model-value="dns01ChallengeRecordValue"
                    readonly
                    auto-grow
                    variant="outlined"
                    density="compact"
                    rows="2"
                />
              </v-col>
              <v-col cols="auto">
                <v-btn class="ml-2 mb-6" color="primary" icon @click="copyToClipboard">
                  <v-icon>mdi-content-copy</v-icon>
                </v-btn>
              </v-col>
            </v-row>
          </v-card>

        </div>
      </v-form>
    </v-card>
    <br>

    <v-card class="pa-6">
      <h2>
        <span>Maintenance</span>
        <DocsReference doc-path="/ocelot-cloud/settings/maintenance"/>
      </h2>
      <v-form @submit.prevent="saveMaintenanceConfigs">
        <v-container>
          <v-checkbox id="auto-updates-enabled-checkbox" v-model="maintenance_auto_updates_enabled" label="Enable Automatic Updates"/>
          <v-checkbox id="auto-backups-enabled-checkbox" v-model="maintenance_auto_backups_enabled" label="Enable Automatic Backups"/>
          <v-select
              id="preferred-hour-selection-dropdown"
              :items="hours"
              item-title="title"
              item-value="value"
              v-model="preferred_maintenance_hour"
              label="Preferred Time For Maintenance (UTC)"
              variant="outlined"
              style="width: 250px; margin-left: 10px"
          />
          <v-btn class="ssh-button" id="maintenance-save-button" color="primary" @click="saveMaintenanceConfigs">Save</v-btn>
        </v-container>
      </v-form>
    </v-card>
    <br>

    <v-card class="pa-6">
      <h2>
        <span>Remote Backup Server</span>
        <DocsReference doc-path="/ocelot-cloud/settings/remote-backup"/>
      </h2>
      <br />
      <v-form>
        <v-container>
          <v-row dense>
            <v-col cols="4" class="column">
              <strong>Remote Host</strong>
            </v-col>
            <ValidationInput
                id="remote-backup-host"
                validationType="host"
                v-model="remote_host"
                :submitted="wasRemoteServerSettingsSubmitted"
                label="enter remote host"
            />
          </v-row>
          <v-row dense>
            <v-col cols="4" class="column">
              <strong>SSH Port</strong>
            </v-col>
            <ValidationInput
                id="remote-backup-ssh-port"
                validationType="number"
                v-model="remote_ssh_port"
                :submitted="wasRemoteServerSettingsSubmitted"
                label="enter port"
            />
          </v-row>
          <v-row dense>
            <v-col cols="4" class="column">
              <strong>SSH User</strong>
            </v-col>
            <ValidationInput
                id="remote-backup-ssh-user"
                validationType="default"
                v-model="remote_ssh_user"
                :submitted="wasRemoteServerSettingsSubmitted"
                label="enter ssh user"
            />
          </v-row>
          <v-row dense>
            <v-col cols="4" class="column">
              <strong>SSH Password</strong>
            </v-col>
            <ValidationInput
                id="remote-backup-ssh-password"
                validationType="password"
                v-model="remote_ssh_password"
                :submitted="wasRemoteServerSettingsSubmitted"
                label="enter ssh password"
                :type="showSshPassword ? 'text' : 'password'"
                :append-inner-icon="showSshPassword ? 'mdi-eye-off' : 'mdi-eye'"
                @click:append-inner="showSshPassword = !showSshPassword"
            />
          </v-row>
          <v-row dense>
            <v-col cols="4" class="column">
              <strong>Data Encryption Password</strong>
            </v-col>
            <ValidationInput
                id="remote-backup-encryption-password"
                validationType="password"
                v-model="remote_encryption_password"
                :submitted="wasRemoteServerSettingsSubmitted"
                label="enter ssh password"
                :type="showEncryptionPassword ? 'text' : 'password'"
                :append-inner-icon="showEncryptionPassword ? 'mdi-eye-off' : 'mdi-eye'"
                @click:append-inner="showEncryptionPassword = !showEncryptionPassword"
            />
          </v-row>
          <v-row dense>
            <v-col cols="4" class="column">
              <strong>SSH Known Hosts</strong>
            </v-col>
            <v-col cols="8">
              <v-textarea
                id="remote-backup-ssh-known-hosts"
                v-model="remote_ssh_known_hosts"
                variant="outlined"
                :auto-grow="showKnownHostsExpanded"
                :append-inner-icon="showKnownHostsExpanded ? '' : 'mdi-chevron-down'"
                @click:append-inner="showKnownHostsExpanded = true"
                required
              />
            </v-col>
          </v-row>
          <v-checkbox id="remote-backup-checkbox" v-model="remote_is_enabled" label="Enable Remote Backups" />
          <v-btn class="ssh-button" id="remote-backup-get-known-hosts" color="primary" @click="getKnownHosts">Get Known Hosts</v-btn>
          <v-btn class="ssh-button" id="remote-backup-test-access" color="primary" @click="testConnection">Test Connection</v-btn>
          <v-btn class="ssh-button" id="remote-backup-save-button" color="primary" @click="confirmSaveRemoteBackupConfigs">Save</v-btn>
        </v-container>
      </v-form>
    </v-card>
    <ConfirmationDialog
        v-model:visible="showSshSettingsSaveConfirmation"
        :on-confirm="saveRemoteBackupConfigs"
        title="Remote Backup Settings Save Confirmation"
        message="You need to be sure that you really trust the 'Known Host'. Do a manual ssh-scan as described in the documentation. If the 'Known Host' does not match your backup server's public SSH keys, you are probably the victim of a man-in-the-middle attack. Are you sure you want to save this configuration?"
    />
  </FrameComponent>
</template>

<script lang="ts">
import {defineComponent, onMounted, ref} from 'vue'
import {doCloudRequest} from "@/components/requests";
import FrameComponent from "@/components/FrameComponent.vue";
import DocsReference from "@/components/DocsReference.vue";
import PageHeader from "@/components/PageHeader.vue";
import {
  backendReadMaintenanceSettingsPath,
  backendReadSshSettingsPath,
  backendSaveMaintenanceSettingsPath, isDemoDomain
} from "@/components/config";
import ValidationInput from "@/components/ValidationInput.vue";
import ConfirmationDialog from "@/components/ConfirmationDialog.vue";

interface RemoteRepositoryData {
    is_enabled: boolean
    host: string
    ssh_port: string
    ssh_user: string
    ssh_password: string
    ssh_known_hosts: string
    encryption_password: string
}

interface MaintenanceSettings {
  are_auto_backups_enabled: boolean
  are_auto_updates_enabled: boolean
  preferred_maintenance_hour: number
}

export default defineComponent({
  name: 'ConfigWizard',
  components: {ConfirmationDialog, ValidationInput, PageHeader, DocsReference, FrameComponent},
  setup() {
    const wasHostSubmitted = ref(false)
    const host = ref("")

    const wasCertificateGenerationRequestSubmitted = ref(false)
    const wasCertificateGenerationRequestSuccessful = ref(false)
    const certificateFileInput = ref<HTMLInputElement | null>(null)
    const certGenerationEmail = ref("")
    const dns01ChallengeRecordName = ref("")
    const dns01ChallengeRecordValue = ref("")

    const wasRemoteServerSettingsSubmitted = ref(false)
    const remote_is_enabled = ref(false)
    const remote_host = ref("")
    const remote_ssh_port = ref("")
    const remote_ssh_user = ref("")
    const remote_ssh_password = ref("")
    const remote_ssh_known_hosts = ref("")
    const remote_encryption_password = ref("")
    const showKnownHostsExpanded = ref(false)
    const showSshPassword = ref<boolean>(false)
    const showEncryptionPassword = ref<boolean>(false)
    const showSshSettingsSaveConfirmation = ref(false)

    const maintenance_auto_backups_enabled = ref(false)
    const maintenance_auto_updates_enabled = ref(false)
    const preferred_maintenance_hour = ref(0)
    const hours = Array.from({ length: 24 }, (_, i) => ({ title: `${i}:00`, value: i }))

    const fetchConfigs = async () => {
      const resp = await doCloudRequest("/api/settings/host/read", null)
      if (resp && resp.status === 200) {
        host.value = resp.data.value
      }
    }

    const saveHost = async () => {
      wasHostSubmitted.value = true
      const resp = await doCloudRequest("/api/settings/host/save", {value: host.value})
      if (resp && resp.status === 200) {
        alert("Host saved successfully")
      }
    }

    function getRemoteRepoDataStructure(): RemoteRepositoryData {
      return {
        is_enabled: remote_is_enabled.value,
        host: remote_host.value,
        ssh_port: remote_ssh_port.value,
        ssh_user: remote_ssh_user.value,
        ssh_password: remote_ssh_password.value,
        ssh_known_hosts: remote_ssh_known_hosts.value,
        encryption_password: remote_encryption_password.value,
      };
    }

    const saveRemoteBackupConfigs = async () => {
      showSshSettingsSaveConfirmation.value = false
      wasRemoteServerSettingsSubmitted.value = true
      let paylod = getRemoteRepoDataStructure()
      const resp = await doCloudRequest("/api/settings/ssh/save", paylod)
      if (resp && resp.status === 200) {
        alert("SSH configs saved successfully")
      }
    }

    const confirmSaveRemoteBackupConfigs = async () => {
      showSshSettingsSaveConfirmation.value = true
    }

    const fetchMaintenanceConfigs = async () => {
      const resp = await doCloudRequest(backendReadMaintenanceSettingsPath, null)
      if (resp && resp.status === 200) {
        maintenance_auto_backups_enabled.value = resp.data.are_auto_backups_enabled
        maintenance_auto_updates_enabled.value = resp.data.are_auto_updates_enabled
        preferred_maintenance_hour.value = resp.data.preferred_maintenance_hour
      }
    }

    const saveMaintenanceConfigs = async () => {
      const resp = await doCloudRequest(backendSaveMaintenanceSettingsPath, getMaintenanceDataStructure())
      if (resp && resp.status === 200) {
        alert("Maintenance configs saved successfully")
      }
    }

    function getMaintenanceDataStructure(): MaintenanceSettings {
      return {
        are_auto_backups_enabled: maintenance_auto_backups_enabled.value,
        are_auto_updates_enabled: maintenance_auto_updates_enabled.value,
        preferred_maintenance_hour: preferred_maintenance_hour.value,
      }
    }

    const fetchRemoteRepositoryConfigs = async () => {
      const resp = await doCloudRequest(backendReadSshSettingsPath, null)
      if (resp && resp.status === 200) {
        remote_is_enabled.value = resp.data.is_enabled
        remote_host.value = resp.data.host
        remote_ssh_port.value = resp.data.ssh_port
        remote_ssh_user.value = resp.data.ssh_user
        remote_ssh_password.value = resp.data.ssh_password
        remote_ssh_known_hosts.value = resp.data.ssh_known_hosts
        remote_encryption_password.value = resp.data.encryption_password
      }
    }

    const handleFileUpload = (event: Event) => {
      const target = event.target as HTMLInputElement;
      if (!target.files || target.files.length === 0) return;

      const file = target.files[0];
      const reader = new FileReader();

      reader.onload = () => {
        if (reader.result instanceof ArrayBuffer) {
          const bytes = new Uint8Array(reader.result);
          processFileBytes(bytes);
          target.value = "";
        }
      };

      reader.readAsArrayBuffer(file);
    };

    const processFileBytes = async (bytes: Uint8Array) => {
      const base64String = btoa(String.fromCharCode(...bytes));
      const resp = await doCloudRequest("/api/certificate/upload", {data: base64String})
      if (resp && resp.status === 200) {
        alert("certificate file uploaded successfully")
      }
    };

    const getKnownHosts = async () => {
      const resp = await doCloudRequest("/api/settings/ssh/known-hosts", getRemoteRepoDataStructure())
      if (resp && resp.status === 200) {
        remote_ssh_known_hosts.value = resp.data.value
        alert("received known hosts successfully")
      }
    }

    const testConnection = async () => {
      const resp = await doCloudRequest("/api/settings/ssh/test-access", getRemoteRepoDataStructure())
      if (resp && resp.status === 200) {
        alert("SSH access test was successful")
      }
    }

    const startDnsChallenge = async () => {
      wasCertificateGenerationRequestSubmitted.value = true
      const resp = await doCloudRequest("/api/settings/certificate/generate", {host: host.value, email: certGenerationEmail.value})
      if (resp && resp.status === 200) {
        alert("Certificate generation request was successful")
        wasCertificateGenerationRequestSuccessful.value = true
        dns01ChallengeRecordName.value = resp.data.host
        dns01ChallengeRecordValue.value = resp.data.base_key_auth + "\n" + resp.data.wildcard_key_auth
      }
    }

    const copyToClipboard = async () => {
      await navigator.clipboard.writeText(dns01ChallengeRecordValue.value)
    }

    onMounted(() => {
      fetchConfigs()
      fetchMaintenanceConfigs()
      fetchRemoteRepositoryConfigs()
    })

    return {
      saveHost,
      host,
      saveRemoteBackupConfigs,
      confirmSaveRemoteBackupConfigs,
      showSshSettingsSaveConfirmation,
      showSshPassword,
      showEncryptionPassword,
      remote_is_enabled,
      remote_host,
      remote_ssh_port,
      remote_ssh_user,
      remote_ssh_password,
      remote_ssh_known_hosts,
      remote_encryption_password,
      handleFileUpload,
      getKnownHosts,
      testConnection,
      showKnownHostsExpanded,
      saveMaintenanceConfigs,
      maintenance_auto_updates_enabled,
      maintenance_auto_backups_enabled,
      preferred_maintenance_hour,
      hours,
      isDemoDomain,
      wasHostSubmitted,
      wasRemoteServerSettingsSubmitted,
      certificateFileInput,
      certGenerationEmail,
      dns01ChallengeRecordName,
      dns01ChallengeRecordValue,
      startDnsChallenge,
      wasCertificateGenerationRequestSubmitted,
      wasCertificateGenerationRequestSuccessful,
      copyToClipboard,
    }
  }
})
</script>

<style lang="sass">
.ssh-button
  margin-right: 15px
.column
  display: flex
  align-items: center
  margin-bottom: 24px
</style>