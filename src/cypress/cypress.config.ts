import { defineConfig } from "cypress";

export default defineConfig({
  e2e: {
    // Add any e2e specific configuration here
  },
  watchForFileChanges: false,
  // should prevent crashes due to out-of-memory errors, which usually occur when running the cypress tests multiple times
  numTestsKeptInMemory: 0,
});
