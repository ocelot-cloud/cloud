<template>
  <FrameComponent>
    <PageHeader title="App Store" doc-path="/ocelot-cloud/app-store"></PageHeader>
    <br>
    <v-row class="text-center" justify="center" v-if="isDemoDomain">
      <v-col cols="8">
        <v-alert type="warning">
          Check "Show Unofficial Apps, enter "demo", search and download apps you do not already have on the home page. Only apps from maintainer "demo" will be displayed in this demo. If you are using your own Ocelot-Cloud instance, you can search for all apps in the App Store.
        </v-alert>
      </v-col>
    </v-row>
    <br>
    <v-row justify="center" align="center">
      <v-col cols="auto" md="5">
        <ValidationInput
            id="search-bar"
            validationType="appSearch"
            v-model="searchQuery"
            :submitted="submitted"
            label="Enter search term for maintainers or apps..."
        />
      </v-col>
    </v-row>
    <v-row justify="center" align="center">
      <v-col cols="auto">
        <v-checkbox
            id="show-unofficial-apps-checkbox"
            v-model="showUnofficialApps"
            label="Show Unofficial Apps"
            hide-details
            density="compact"
        />
      </v-col>
      <v-col cols="auto">
        <v-btn
            id="search-button"
            color="primary"
            @click="executeSearch"
        >
          <v-icon left>mdi-magnify</v-icon> Search
        </v-btn>
      </v-col>
    </v-row>

    <v-row>
      <v-col cols="12" v-if="filteredApps.length > 0">
        <v-data-table
            :headers="headers"
            :items="filteredApps"
            item-key="app_id"
            dense
            outlined
        >
          <template v-slot:body="{ items }">
            <tr
                v-for="item in items"
                :key="item.app_id"
            >
              <td class="maintainer-column">{{ item.maintainer }}</td>
              <td class="name-column">{{ item.app_name }}</td>
              <td class="latest-version-column text-center">
                <!-- Removed @click="fetchVersions(item)" so we can fetch all versions on search -->
                <v-select
                    :items="versionsMap[item.app_id] || []"
                    item-title="name"
                    item-value="id"
                    v-model="selectedVersions[item.app_id]"
                    placeholder="Select version"
                    hide-details
                />
              </td>
              <td class="text-center">
                <v-btn
                    id="install-button"
                    color="primary"
                    @click="installApp(item)"
                >
                  Install
                </v-btn>
                <!-- TODO: 1) to be tested; 2) why can't I unzip it on my PC? bad formatting? -->
                <v-btn
                    id="download-button"
                    color="primary"
                    @click="downloadApp(item)"
                    style="min-width: 36px; height: 36px; margin-left: 10px;"
                >
                  <v-icon>mdi-download</v-icon>
                </v-btn>

              </td>
            </tr>
          </template>
        </v-data-table>
      </v-col>
      <v-col class="text-center" cols="12" v-else>
        <p>Please search for apps.</p>
      </v-col>
    </v-row>
  </FrameComponent>
</template>

<script lang="ts">
import { defineComponent, ref, computed } from 'vue';
import { useRouter } from 'vue-router';
import { doCloudRequest } from '@/components/requests';
import { frontendHomePath, isDemoDomain } from '@/components/config';
import FrameComponent from "@/components/FrameComponent.vue";
import PageHeader from "@/components/PageHeader.vue";
import ValidationInput from "@/components/ValidationInput.vue";

interface App {
  maintainer: string;
  app_id: string;
  app_name: string;
  latest_version_id: string;
  latest_version_name: string;
}

interface Version {
  id: string;
  name: string;
  version_creation_timestamp: string;
}

export default defineComponent({
  name: 'AppStore',
  components: {ValidationInput, PageHeader, FrameComponent },
  setup() {
    const apps = ref<App[]>([]);
    const searchQuery = ref('');
    const submitted = ref(false);
    const router = useRouter();
    const showUnofficialApps = ref(false);

    // Store all versions for each app, keyed by app_id
    const versionsMap = ref<{ [key: string]: Version[] }>({});

    // Store the currently selected version ID for each app
    const selectedVersions = ref<{ [key: string]: string }>({});

    const headers = ref([
      { title: 'Maintainer', key: 'maintainer', align: 'start' },
      { title: 'App', key: 'app_name', align: 'start' },
      { title: 'Version', key: 'latest_version_name', align: 'center' },
      { title: 'Actions', key: 'action', align: 'center' },
    ]);

    const fetchVersions = async (app: App) => {
      if (!versionsMap.value[app.app_id]) {
        const resp = await doCloudRequest('/api/versions/list', { value: app.app_id });
        if (resp && resp.status === 200 && Array.isArray(resp.data)) {
          versionsMap.value[app.app_id] = resp.data.sort(
              (a, b) => new Date(b.version_creation_timestamp).getTime() - new Date(a.version_creation_timestamp).getTime()
          );
        }
      }
    };

    const installApp = async (app: App) => {
      const versionId =
          selectedVersions.value[app.app_id] || app.latest_version_id;
      const resp = await doCloudRequest('/api/versions/install', { value: versionId });
      if (resp && resp.status === 200) {
        alert('Installation successful. Visit the app on the Installed Apps page.');
      }
    };

    const downloadApp = async (app: App) => {
      const versionId = selectedVersions.value[app.app_id] || app.latest_version_id;
      const resp = await doCloudRequest('/api/versions/download', { value: versionId });
      if (resp && resp.status === 200) {
        const b64 = resp.data.content; // base-64 string
        const bytes = Uint8Array.from(atob(b64), c => c.charCodeAt(0));
        const blob = new Blob([bytes], { type: 'application/zip' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = resp.data.file_name;
        a.click();
        URL.revokeObjectURL(url);
      }
    };

    const executeSearch = async () => {
      submitted.value = true

      let data = {
        search_term: searchQuery.value,
        show_unofficial_apps: showUnofficialApps.value,
      }
      const resp = await doCloudRequest('/api/apps/search', data);
      if (resp && resp.status) {
        apps.value = resp.data;

        // For each found app:
        // 1. Set the default version ID (so "2.0" shows by default)
        // 2. Pre-fetch the versions so that the v-select can display the proper name
        for (const app of apps.value) {
          selectedVersions.value[app.app_id] = app.latest_version_id;
          await fetchVersions(app);
        }
      } else {
        apps.value = []
      }
    };

    const filteredApps = computed(() => {
      return isDemoDomain.value
          ? apps.value.filter(app => app.maintainer === 'demo')
          : apps.value;
    });

    return {
      apps,
      searchQuery,
      headers,
      installApp,
      executeSearch,
      filteredApps,
      frontendHomePath,
      router,
      isDemoDomain,
      versionsMap,
      selectedVersions,
      fetchVersions,
      showUnofficialApps,
      submitted,
      downloadApp,
    };
  },
});
</script>
