import axios from "axios";
import sessionStore from "./sessionStore.js";

const instance = axios.create({
	baseURL: __API_URL__,
	timeout: 1000 * 10
});

// Attach Authorization header when a token is available.
instance.interceptors.request.use((config) => {
	if (sessionStore.state.token) {
		config.headers.Authorization = `Bearer ${sessionStore.state.token}`;
	}
	return config;
});

export default instance;
