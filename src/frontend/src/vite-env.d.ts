// Without this file, the Jetbrains IDE will complain: "Vue: Property env does not exist on type ImportMeta" even though everything works.
interface ImportMetaEnv {
    readonly VITE_BASE_URL: string
    readonly VITE_APP_PROFILE: string
}

interface ImportMeta {
    readonly env: ImportMetaEnv
}