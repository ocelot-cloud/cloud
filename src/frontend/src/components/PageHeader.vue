<template>
  <div class="text-center">
    <h1>
      <span>{{ title }}</span>
      <DocsReference :doc-path="docPath" />
    </h1>
  </div>
</template>

<script setup lang="ts">
import { defineProps} from 'vue'
import DocsReference from "@/components/DocsReference.vue";
defineProps<{
  title: string
  docPath: string
}>()
</script>
