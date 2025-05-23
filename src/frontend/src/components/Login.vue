<template>
  <v-app>
    <v-container>
      <v-row align="center" justify="center">
        <v-col cols="12" md="5">
          <v-card>
            <v-card-title class="text-center">Login</v-card-title>
            <v-card-text>
              <v-row class="text-center" justify="center" v-if="isDemoDomain">
                <v-col cols="12">
                  <v-alert type="warning">
                    This is the demo version of Ocelot-Cloud. The username is "admin" and password is "password".
                  </v-alert>
                </v-col>
              </v-row>
              <br>
              <v-form @submit.prevent="login" class="mt-1">
                <ValidationInput
                    id="username-field"
                    validationType="default"
                    v-model="username"
                    :submitted="submitted"
                    label="Username"
                />
                <ValidationInput
                    id="password-field"
                    validationType="password"
                    v-model="password"
                    :submitted="submitted"
                    label="Password"
                />
                <v-btn type="submit" color="primary" id="login-button">Login</v-btn>
              </v-form>
            </v-card-text>
          </v-card>
        </v-col>
      </v-row>
    </v-container>
  </v-app>
</template>


<script lang="ts">
import { defineComponent, ref } from 'vue'
import { goToPage } from '@/components/shared'
import { doCloudRequest } from '@/components/requests'
import {isDemoDomain} from "@/components/config";
import ValidationInput from "@/components/ValidationInput.vue";

export default defineComponent({
  name: 'login-component',
  components: {ValidationInput},
  setup() {
    const username = ref('')
    const password = ref('')
    const submitted = ref(false)

    const login = async () => {
      const loginForm = { username: username.value, password: password.value }
      const response = await doCloudRequest('/api/login', loginForm)
      if (response) {
        goToPage('/')
      } else {
        submitted.value = true
      }
    }

    return {
      username,
      password,
      login,
      isDemoDomain,
      submitted,
    }
  }
})
</script>
