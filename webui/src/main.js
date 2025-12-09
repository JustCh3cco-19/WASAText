import {createApp} from 'vue';
import App from './App.vue';
import router from './router';
import axios from './services/axios.js';
import sessionStore from './services/sessionStore.js';
import ErrorMsg from './components/ErrorMsg.vue';
import LoadingSpinner from './components/LoadingSpinner.vue';

import './assets/dashboard.css';
import './assets/main.css';

const app = createApp(App);
app.config.globalProperties.$axios = axios;
app.config.globalProperties.$session = sessionStore;
app.provide("session", sessionStore);
app.component("ErrorMsg", ErrorMsg);
app.component("LoadingSpinner", LoadingSpinner);
app.use(router);
app.mount('#app');
