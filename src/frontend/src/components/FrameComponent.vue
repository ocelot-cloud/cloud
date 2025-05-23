<template>
  <v-app>
    <v-app-bar flat>
      <v-row align="center" justify="space-between" class="pa-2">
        <v-col cols="auto">
          <img src="../assets/logo.png" alt="Logo" style="height: 45px;" />
        </v-col>
        <v-col class="white--text text-h5">
          Ocelot-Cloud
        </v-col>
      </v-row>
    </v-app-bar>
    <v-navigation-drawer app v-model="drawer" permanent>
      <v-toolbar flat>
        <v-toolbar-title>Navigation</v-toolbar-title>
      </v-toolbar>
      <v-divider></v-divider>
      <v-list>
        <v-list-item id="go-to-installed-apps" @click="router.push(frontendHomePath)" prepend-icon="mdi-apps">
          <v-list-item-title>Installed Apps</v-list-item-title>
        </v-list-item>
        <v-list-item id="go-to-store" v-if="cloudSession.isAdmin" @click="router.push(frontendStorePath)" prepend-icon="mdi-store">
          <v-list-item-title>App Store</v-list-item-title>
        </v-list-item>
        <v-list-item id="go-to-users" v-if="cloudSession.isAdmin" @click="router.push(frontendUsersPath)" prepend-icon="mdi-account-multiple">
          <v-list-item-title>Users</v-list-item-title>
        </v-list-item>
        <v-list-item id="go-to-backups" v-if="cloudSession.isAdmin" @click="router.push(frontendBackupsPath)" prepend-icon="mdi-database-arrow-down">
          <v-list-item-title>Backups</v-list-item-title>
        </v-list-item>
        <v-list-item id="go-to-settings" v-if="cloudSession.isAdmin && !isDemoDomain" @click="router.push(frontendSettingsPath)" prepend-icon="mdi-cog">
          <v-list-item-title>Settings</v-list-item-title>
        </v-list-item>
        <v-list-item id="change-password" @click="router.push(frontendChangePasswordPath)" prepend-icon="mdi-lock-reset">
          <v-list-item-title>Change Password</v-list-item-title>
        </v-list-item>
        <v-list-item id="logout" @click="logout()" prepend-icon="mdi-logout">
          <v-list-item-title>Logout</v-list-item-title>
        </v-list-item>
      </v-list>
      <v-divider></v-divider>
      <div class="pa-4 text-center">
        Logged in as: <strong>{{ cloudSession.user }}</strong>
      </div>
    </v-navigation-drawer>

    <v-main>
      <v-container>
        <slot />
      </v-container>
    </v-main>
  </v-app>


</template>

<script lang="ts">
import {defineComponent, ref} from 'vue'
import { useRouter } from 'vue-router'
import {
  cloudSession,
  frontendStorePath,
  frontendUsersPath,
  frontendLoginPath,
  frontendHomePath,
  frontendSettingsPath,
  frontendVersionsPath,
  frontendBackupsPath,
  frontendAboutPath, frontendRepositoriesPath, isDemoDomain, frontendChangePasswordPath
} from '@/components/config'
import { doCloudRequest } from '@/components/requests'

export default defineComponent({
  name: 'frame-component',
  setup() {
    const drawer = ref(true)
    const router = useRouter()

    const logout = async () => {
      let response = await doCloudRequest('/api/users/logout', null)
      if (response && response.status === 200) {
        cloudSession.user = ''
        cloudSession.isAdmin = false
        cloudSession.isAuthenticated = false
        router.push(frontendLoginPath)
      }
    }

    return {
      drawer,
      router,
      logout,
      cloudSession,
      frontendStorePath,
      frontendUsersPath,
      frontendHomePath,
      frontendSettingsPath,
      frontendVersionsPath,
      frontendBackupsPath,
      frontendAboutPath,
      frontendChangePasswordPath,
      frontendRepositoriesPath,
      isDemoDomain,
    }
  }
})
</script>

<style scoped lang="sass">
.v-navigation-drawer
  width: 256px
</style>