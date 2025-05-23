<template>
  <FrameComponent>
    <div class="text-center">
      <h1>
        <span>Change Password</span>
      </h1>
    </div>
    <br>
    <v-container>
      <v-row align="center" justify="center">
        <v-col cols="12" md="5">
          <v-card>
            <v-card-text>
              <br>
              <v-form @submit.prevent="changePassword">
                <ValidationInput
                    id="password-field"
                    validationType="password"
                    v-model="password"
                    :submitted="submitted"
                    label="New Password"
                />
                <v-btn type="submit" color="primary" id="change-password-button" class="mt-1">Change Password</v-btn>
              </v-form>
            </v-card-text>
          </v-card>
        </v-col>
      </v-row>
    </v-container>
  </FrameComponent>
</template>


<script lang="ts">
import {defineComponent, ref} from "vue"
import { doCloudRequest } from "@/components/requests"
import FrameComponent from "@/components/FrameComponent.vue";
import ValidationInput from "@/components/ValidationInput.vue";

export default defineComponent({
  name: "ChangePasswordComponent",
  components: {FrameComponent, ValidationInput},
  setup() {
    const password = ref("")
    const submitted = ref(false)

    const changePassword = async () => {
      submitted.value = true
      const response = await doCloudRequest('/api/users/change-password', {value: password.value})
      if (response && response.status === 200) {
        alert("Password changed successfully")
      }
    }

    return {
      password,
      changePassword,
      submitted,
    }
  }
})
</script>

