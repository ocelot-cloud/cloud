<template>
  <v-dialog v-model="visible" max-width="500px">
    <v-card>
      <v-card-title class="text-h6">{{ title }}</v-card-title>
      <v-card-text>
        <p>{{ message }}</p>
      </v-card-text>
      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn
            v-if="onConfirm"
            id="button-confirmation"
            color="error"
            @click="onConfirm"
        >
          Confirm
        </v-btn>
        <v-btn
            id="button-close-popup"
            color="secondary"
            @click="closeModal"
        >
          Cancel
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script lang="ts">
import { defineComponent, PropType } from 'vue'

export default defineComponent({
  props: {
    visible: {
      type: Boolean,
      required: true
    },
    title: {
      type: String,
      required: true
    },
    message: {
      type: String,
      required: true
    },
    onConfirm: {
      type: Function as PropType<(event: MouseEvent) => void>,
      required: false
    }
  },
  methods: {
    closeModal() {
      this.$emit('update:visible', false)
    }
  }
})
</script>
