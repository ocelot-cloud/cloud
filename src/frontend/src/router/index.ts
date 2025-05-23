import { createRouter, createWebHistory } from 'vue-router';
import Home from "@/components/Home.vue";
import Login from "@/components/Login.vue";
import axios from "axios";
import UserManagement from "@/components/UserManagement.vue";
import {cloudBaseUrl, cloudSession, frontendChangePasswordPath, Session} from "@/components/config";
import AppStore from "@/components/AppStore.vue";
import SettingsComponent from "@/components/SettingsComponent.vue";
import BackupsComponent from "@/components/BackupsComponent.vue";
import AboutComponent from "@/components/AboutComponent.vue";
import InstalledApps from "@/components/InstalledApps.vue";
import ChangePasswordComponent from "@/components/ChangePasswordComponent.vue";

const routes = [
    {
        path: '/',
        name: 'InstalledApps',
        component: InstalledApps,
    },
    {
        path: '/store',
        name: 'AppStore',
        component: AppStore,
    },
    {
        path: '/users',
        name: 'UserManagement',
        component: UserManagement,
    },
    {
        path: '/login',
        name: 'Login',
        component: Login,
    },
    {
        path: '/settings',
        name: "SettingsComponent",
        component: SettingsComponent,
    },
    {
        path: '/backups',
        name: "BackupsComponent",
        component: BackupsComponent,
    },
    {
        path: '/' + frontendChangePasswordPath,
        name: "ChangePasswordComponent",
        component: ChangePasswordComponent,
    },
];

const router = createRouter({
    history: createWebHistory(import.meta.env.VITE_BASE_URL),
    routes,
});

router.beforeEach(async (to, from, next) => {
    if (to.path === '/login' || cloudSession.isAuthenticated) {
        next();
    } else {
        const isSessionValid = await isThereValidSession(cloudSession, cloudBaseUrl + '/api/check-auth');
        if (isSessionValid) {
            next();
        } else {
            next({ name: 'Login' });
        }
    }
});

async function isThereValidSession(session: Session, url: string): Promise<boolean> {
    try {
        const response = await axios.get(url);
        if (response.status === 200) {
            session.user = response.data.user;
            session.isAdmin = response.data.is_admin;
            session.isAuthenticated = true;
            return true
        } else {
            return false
        }
    } catch (error) {
        return false
    }
}

export default router;
