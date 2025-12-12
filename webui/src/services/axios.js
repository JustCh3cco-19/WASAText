import axios from "axios";
import sessionStore from "./sessionStore.js";

const resolveApiBaseUrl = () => {
	const configured = (__API_URL__ || "").trim();
	const isLocalhost =
		configured.startsWith("http://localhost") ||
		configured.startsWith("https://localhost") ||
		configured.startsWith("http://127.0.0.1") ||
		configured.startsWith("https://127.0.0.1");

	// Respect explicit non-local configuration.
	if (configured && !isLocalhost) {
		return configured;
	}

	// If we're in the browser, use the current host with the backend port.
	if (typeof window !== "undefined") {
		const {protocol, hostname} = window.location;
		return `${protocol}//${hostname}:3000`;
	}

	// Fallback for non-browser contexts.
	return configured || "http://localhost:3000";
};

const instance = axios.create({
	baseURL: resolveApiBaseUrl(),
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
