import axios, {AxiosRequestConfig, AxiosResponse} from "axios";
import {cloudBaseUrl} from "@/components/config";

export function alertError(error: any) {
    if (axios.isAxiosError(error) && error.response) {
        const errorMessage = error.response.data || 'An unknown error occurred';
        alert(`An error occurred: ${errorMessage}`);
    } else {
        alert('An unknown error occurred');
    }
}

export async function doRequest(url: string, data: any): Promise<(AxiosResponse | null)> {
    try {
        const response = await axios.post(url, data, {
            withCredentials: true,
        });
        if (response.status !== 200 && response.status !== 204) {
            throw new Error(response.data);
        }
        return response
    } catch (error: any) {
        // This try-catch block is necessary to prevent the user from being disturbed by NETWORK_CHANGED errors in the browser. Whenever an app is started for the first time, the ocelotcloud container will connect to the app-specific Docker network. This changes the network and causes the error. The same happens when an app is deleted, since the ocelotcloud container will disconnect from that app-specific Docker network, also changing its network.
        if (axios.isAxiosError(error) && (!error.response || error.message?.includes('NETWORK_CHANGED'))) {
            return {
                data: {},
                status: 200,
                statusText: 'OK',
                headers: {},
                config: {} as AxiosRequestConfig,
            } as AxiosResponse
        } else {
            alertError(error);
            return null
        }
    }
}

export async function doCloudRequest(path: string, body: any) {
    return await doRequest(cloudBaseUrl + path, body)

}