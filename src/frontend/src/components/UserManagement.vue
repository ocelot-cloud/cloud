<template>
  <FrameComponent>
    <PageHeader title="Users" doc-path="/ocelot-cloud/users"></PageHeader>
    <br>
    <div class="d-flex justify-center align-center">
      <v-form @submit.prevent="createUser" class="d-flex" style="gap: 16px">
        <ValidationInput
            style="width: 250px;"
            id="username"
            validationType="default"
            v-model="usernameField"
            :submitted="submitted"
            label="Enter username"
        />
        <ValidationInput
            style="width: 250px;"
            id="password"
            validationType="password"
            v-model="passwordField"
            :submitted="submitted"
            label="Enter password"
        />
        <v-btn id="submit" type="submit" color="primary" style="margin-top: 10px">Create User</v-btn>
      </v-form>
    </div>

    <v-data-table
        id="user-list"
        :headers="headers"
        :items="users"
        item-value="id"
        class="elevation-1"
    >
      <template v-slot:body="{ items }">
        <tr v-for="item in items" :key="item.id">
          <td class="name-column">{{ item.name }}</td>
          <td class="role-column">{{ item.role }}</td>
          <td>
            <v-btn id="delete-user-button" color="error" @click="confirmDeleteUser(item.id)" :disabled="cloudSession.user === item.name">Delete</v-btn>
          </td>
        </tr>
      </template>
    </v-data-table>
    <ConfirmationDialog
        v-model:visible="showConfirmation"
        :on-confirm="deleteUser"
        title="User Deletion Confirmation"
        message="Are you sure you want to delete this user?"
    />
  </FrameComponent>
</template>

<script lang="ts">
import { defineComponent, ref, onMounted, reactive } from 'vue'
import { useRouter } from 'vue-router'
import {cloudSession, frontendHomePath} from '@/components/config'
import { doCloudRequest } from '@/components/requests'
import FrameComponent from "@/components/FrameComponent.vue";
import PageHeader from "@/components/PageHeader.vue";
import ValidationInput from "@/components/ValidationInput.vue";
import ConfirmationDialog from "@/components/ConfirmationDialog.vue";

interface User {
  id: string
  name: string
  role: string
}

export default defineComponent({
  name: 'UserManagement',
  components: {ConfirmationDialog, ValidationInput, PageHeader, FrameComponent},
  setup() {
    const users = ref<User[]>([])
    const usernameField = ref('')
    const passwordField = ref('')
    const showConfirmation = ref(false)

    const userToDelete = ref('')

    const router = useRouter()
    const submitted = ref(false)

    const headers = [
      { title: 'Name', key: 'name' },
      { title: 'Role', key: 'role' },
      { title: 'Action', key: 'action', sortable: false }
    ]

    const fetchUsers = async () => {
      const resp = await doCloudRequest('/api/users/list', null)
      if (resp && resp.status === 200) {
        users.value = resp.data
      }
    }

    const createUser = async () => {
      submitted.value = true
      const resp = await doCloudRequest('/api/users/create', { username: usernameField.value, password: passwordField.value })
      if (resp && resp.status === 200) {
        usernameField.value = ''
        passwordField.value = ''
        submitted.value = false
        fetchUsers()
      }
    }

    const deleteUser = async () => {
      showConfirmation.value = false
      const resp = await doCloudRequest('/api/users/delete', { value: userToDelete.value })
      if (resp && resp.status === 200) {
        fetchUsers()
      }
    }

    const confirmDeleteUser = async (name: string) => {
      userToDelete.value = name
      showConfirmation.value = true
    }

    onMounted(() => {
      fetchUsers()
    })

    return {
      users,
      usernameField,
      passwordField,
      createUser,
      confirmDeleteUser,
      deleteUser,
      router,
      frontendHomePath,
      headers,
      cloudSession,
      submitted,
      showConfirmation,
    }
  }
})
</script>
