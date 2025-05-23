import {ref} from "vue";

export let cloudBaseUrl: string;
export let protocol : string
export let ocelotCloudHost : string
export let port: string

function getGlobalConfig() {
    const PROFILE = import.meta.env.VITE_APP_PROFILE || PROFILE_VALUES.DOCKER;
    if (PROFILE === PROFILE_VALUES.NATIVE) {
        protocol = 'http:';
        ocelotCloudHost = 'localhost';
        port = '8080';
    } else if (PROFILE === PROFILE_VALUES.DOCKER) {
        protocol = window.location.protocol;
        ocelotCloudHost = window.location.hostname
        port = window.location.port;
    } else {
        throw new Error("error, provided VITE_APP_PROFILE is not known: " + PROFILE);
    }
    cloudBaseUrl = protocol + '//' + ocelotCloudHost + (port ? ':' + port : '');
}

getGlobalConfig()

enum PROFILE_VALUES {
    NATIVE = "NATIVE",
    DOCKER = "DOCKER",
}

export const backendReadSshSettingsPath = "/api/settings/ssh/read"
export const backendReadMaintenanceSettingsPath = "/api/settings/maintenance/read"
export const backendSaveMaintenanceSettingsPath = "/api/settings/maintenance/save"

export const frontendHomePath = "/"
export const frontendStorePath = "store"
export const frontendUsersPath = "users"
export const frontendVersionsPath = "versions"
export const frontendBackupsPath = "backups"
export const frontendLoginPath = "login"
export const frontendSettingsPath = "settings"
export const frontendAboutPath = "about"
export const frontendChangePasswordPath = "change-password"
export const frontendRepositoriesPath = "repositories"
export var isDemoDomain = ref(window.location.origin === 'https://demo.ocelot-cloud.org' || window.location.origin === 'https://test.ocelot-cloud.org')

export function switchIsDemoDomain() {
    isDemoDomain.value = !isDemoDomain.value
}

export interface Session {
    user: string;
    isAdmin: boolean;
    isAuthenticated: boolean;
}

export const cloudSession: Session = {
    user: "",
    isAdmin: false,
    isAuthenticated: false,
};