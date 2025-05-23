<template>
  <a :href="fullUrl" target="_blank">
    <v-icon icon="mdi-information-outline" class="ml-2" style="cursor: pointer" />
  </a>
</template>

<script setup lang="ts">
import {computed, defineProps} from 'vue'
const props = defineProps<{
  docPath: string
}>()
const fullUrl = computed(() => `https://ocelot-cloud.org/docs${props.docPath}`)
</script>
