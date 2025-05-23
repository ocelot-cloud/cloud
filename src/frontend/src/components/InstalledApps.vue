<template>
  <FrameComponent>
    <PageHeader title="Installed Apps" doc-path="/ocelot-cloud/installed-apps"></PageHeader>
    <br>
    <v-row class="text-center" justify="center" v-if="isDemoDomain">
      <v-col cols="12">
        <v-alert type="warning">
          Here you can apply operations on apps installed via the App Store. The ocelotdb is the internal database, so only backups are allowed as operations. Starting an app switches its status to available, but in actually may take a few seconds to minutes until the web interface is actually available via "Open" button, as the app need to be downloaded and started.
        </v-alert>
      </v-col>
    </v-row>
    <br>
    <v-data-table
        id="app-list"
        :headers="filteredHeaders"
        :items="apps"
        item-key="app_id"
        dense
        outlined
        style=""
    >
      <template v-slot:body="{ items }">
        <tr v-for="item in items" :key="item.app_id">
          <td v-if="cloudSession.isAdmin">{{ item.maintainer }}</td>
          <td>{{ item.app_name }}</td>
          <td class="version-name-cell text-center" v-if="cloudSession.isAdmin">{{ item.version_name }}</td>
          <td class="text-center">
            <v-btn
                id="open-button"
                color="primary"
                @click="openAppLink(item)"
                :disabled="isOcelotDb(item)"
            >
              <v-icon left>mdi-open-in-new</v-icon> Open
            </v-btn>
          </td>
          <td v-if="cloudSession.isAdmin" class="text-center">
            <v-menu>
              <template v-slot:activator="{ props }">
                <v-btn id="operations-dropdown" color="primary" v-bind="props" icon>
                  <v-icon>mdi-cog</v-icon>
                </v-btn>
              </template>
              <v-list>
                <v-list-item
                    v-if="!(isOcelotDb(item))"
                    id="start-app-button"
                    @click="startApp(item.app_id)"
                >
                  <v-icon left>mdi-play-circle</v-icon> Start
                </v-list-item>
                <v-list-item
                    v-if="!(isOcelotDb(item))"
                    id="stop-app-button"
                    @click="stopApp(item.app_id)"
                >
                  <v-icon left>mdi-stop-circle</v-icon> Stop
                </v-list-item>
                <v-list-item id="backup-app-button" @click="createBackup(item.app_id)">
                  <v-icon left>mdi-database-arrow-down</v-icon> Backup
                </v-list-item>
                <v-list-item
                    v-if="!(isOcelotDb(item))"
                    id="update-app-button"
                    @click="updateApp(item.app_id)"
                >
                  <v-icon left>mdi-update</v-icon> Update
                </v-list-item>
                <v-list-item
                    v-if="!(isOcelotDb(item))"
                    id="prune-app-button"
                    @click="confirmDeleteApp(item.app_id)"
                >
                  <v-icon left>mdi-delete</v-icon> Delete
                </v-list-item>
              </v-list>
            </v-menu>
          </td>
          <td v-if="cloudSession.isAdmin" class="status-column text-center" :class="{ 'text-success': item.status === 'Available', 'text-warning': item.status === 'Uninitialized' }">
            {{ item.status }}
          </td>
        </tr>
      </template>
    </v-data-table>
    <v-col class="text-center mt-5" cols="12"
           v-if = "apps.length == 0">
      <p>No apps found. Please visit the app store.</p>
    </v-col>
    <ConfirmationDialog
        v-model:visible="showConfirmation"
        :on-confirm="deleteApp"
        title="App Deletion Confirmation"
        message="This action will remove all persistent data of that app. Backups will not be affected. Are you sure you want to delete this app?"
    />
  </FrameComponent>
</template>


<script lang="ts">
import {defineComponent, ref, onMounted, computed} from "vue"
import { useRouter } from "vue-router"
import { doCloudRequest } from "@/components/requests"
import {
  frontendStorePath,
  frontendUsersPath,
  protocol,
  cloudSession,
  frontendLoginPath, isDemoDomain
} from "@/components/config"
import FrameComponent from "@/components/FrameComponent.vue";
import {AppDto} from "@/components/shared";
import DocsReference from "@/components/DocsReference.vue";
import PageHeader from "@/components/PageHeader.vue";
import ConfirmationDialog from "@/components/ConfirmationDialog.vue";

export default defineComponent({
  name: "home-component",
  components: {ConfirmationDialog, PageHeader, DocsReference, FrameComponent},
  methods: {
    frontendUsersPath() { return frontendUsersPath },
    frontendStorePath() { return frontendStorePath }
  },
  setup() {
    const host = ref("")
    const apps = ref<AppDto[]>([])
    const headers = [
      { title: "Maintainer", key: "maintainer", show: cloudSession.isAdmin, align: "start"},
      { title: "Name", key: "app_name", show: true, align: "start" },
      { title: "Current Version", key: "version_name", show: cloudSession.isAdmin, align: "center" },
      { title: "Link", key: "link", show: true, align: "center" },
      { title: "Operation", key: "operation", show: cloudSession.isAdmin, align: "center" },
      { title: "Status", key: "status", show: cloudSession.isAdmin, align: "center" }
    ]

    const idOfAppToDelete = ref("")
    const showConfirmation = ref(false)

    const filteredHeaders = computed(() => headers.filter(header => header.show));

    const router = useRouter()

    const fetchApps = async () => {
      const response = await doCloudRequest("/api/apps/list", null);
      if (response && response.status === 200) {
        const data: AppDto[] = Array.isArray(response.data) ? response.data : [];
        data.forEach(app => {
          if (app.maintainer === "ocelotcloud" && app.app_name === "ocelotdb") {
            app.status = "Available";
          }
        });
        apps.value = data.filter(
            (app: AppDto) =>
                cloudSession.isAdmin ||
                (app.status === "Available" &&
                    !isOcelotDb(app))
        );
      }
    };

    const openAppLink = async (app: AppDto) => {
      if (host.value == "") {
        alert("you must set the 'host' in the settings before you can open apps")
        return
      }
      let resp = await doCloudRequest("/api/secret", null)
      if (resp && resp.status === 200) {
        window.open(
            protocol + "//" + app.app_name + "." + host.value + app.url_path + "?ocelot-secret=" + resp.data,
            "_blank"
        )
      }
    }

    const startApp = async (app_id: string) => {
      let response = await doCloudRequest("/api/apps/start", { value: app_id })
      if (response && response.status === 200) {
        alert("App started successfully")
      }
      await fetchApps()
    }

    const stopApp = async (app_id: string) => {
      let response = await doCloudRequest("/api/apps/stop", { value: app_id })
      if (response && response.status === 200) {
        alert("App stopped successfully")
        fetchApps()
      } else if (response) {
        console.log("error: ", response.data)
      }
      await fetchApps()
    }

    const createBackup = async (app_id: string) => {
      let response = await doCloudRequest("/api/backups/create", { value: app_id })
      if (response && response.status === 200) {
        alert("Backup created successfully")
      }
      await fetchApps()
    }

    const updateApp = async (app_id: string) => {
      let response = await doCloudRequest("/api/apps/update", { value: app_id })
      if (response && response.status === 200) {
        alert("App updated successfully")
      }
      await fetchApps()
    }

    const deleteApp = async () => {
      showConfirmation.value = false
      let response = await doCloudRequest("/api/apps/prune", { value: idOfAppToDelete.value })
      if (response && response.status === 200) {
        alert("App deleted successfully")
      }
      await fetchApps()
    }

    const confirmDeleteApp = async (app_id: string) => {
      idOfAppToDelete.value = app_id
      showConfirmation.value = true
    }

    const isOcelotDb = (app: AppDto) => {
      return app.maintainer === "ocelotcloud" && app.app_name === "ocelotdb"
    }

    const logout = async () => {
      let response = await doCloudRequest("/api/users/logout", null)
      if (response && response.status === 200) {
        cloudSession.user = ""
        cloudSession.isAdmin = false
        cloudSession.isAuthenticated = false
        router.push(frontendLoginPath)
      }
    }

    const fetchHost = async () => {
      const resp = await doCloudRequest("/api/settings/host/read", null)
      if (resp && resp.status === 200) {
        host.value = resp.data.value
        console.log("Fetched Host: " + host.value)
      }
    }

    onMounted(fetchApps)
    onMounted(fetchHost)

    return {
      apps,
      filteredHeaders,
      router,
      openAppLink,
      startApp,
      stopApp,
      deleteApp,
      confirmDeleteApp,
      logout,
      cloudSession,
      createBackup,
      updateApp,
      isOcelotDb,
      isDemoDomain,
      showConfirmation,
    }
  }
})
</script>

