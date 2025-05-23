import router from "@/router";

export function goToPage(path: string) {
    router.push(path)
}

export interface AppDto {
    maintainer: string
    app_name: string
    app_id: string
    version_name: string
    url_path: string
    status: string
}