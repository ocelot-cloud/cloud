<template>
  <div id="app">
    <router-view></router-view>
  </div>
</template>

<script lang="ts">
import { Options, Vue } from 'vue-class-component';
import InstalledApps from './components/InstalledApps.vue';
import axios from "axios";

@Options({
  components: {
    InstalledApps,
  },
})
export default class App extends Vue {

}

axios.defaults.withCredentials = true;
</script>
