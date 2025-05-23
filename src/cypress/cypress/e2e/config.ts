export const ocelotCookie = 'ocelot-auth'
export let frontendRootDomain = "localhost:8081"
export let ocelotUrl = "http://" + frontendRootDomain
export const sampleAppName = "sampleapp"
export const sampleMaintainer = "samplemaintainer"
export var cookie = ""
export var ocelotDbMaintainer = "ocelotcloud"
export var ocelotDbName = "ocelotdb"

export function setCookie(newCookie : string) {
    cookie = newCookie
}
