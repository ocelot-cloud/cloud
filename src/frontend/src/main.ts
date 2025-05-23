import { createApp } from 'vue';
import App from './App.vue';
import router from "@/router";

import 'vuetify/styles';
import '@mdi/font/css/materialdesignicons.css';
import { createVuetify } from 'vuetify';
import * as components from 'vuetify/components';
import * as directives from 'vuetify/directives';
import { mdi } from 'vuetify/iconsets/mdi';
import {installKeyboardListener} from "@/components/keyboardListener";

const vuetify = createVuetify({
    theme: {
        defaultTheme: 'dark',
    },
    icons: {
        defaultSet: 'mdi',
        sets: {
            mdi,
        },
    },
    components,
    directives,
});

installKeyboardListener();

createApp(App).use(vuetify).use(router).mount('#app');
