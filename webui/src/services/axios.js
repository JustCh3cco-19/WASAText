import axios from "axios";
import sessionStore from "./sessionStore.js";

const resolveApiBaseUrl = () => {
	const configured = (__API_URL__ || "").trim();
	const parsed = configured ? (() => {
		try {
			return new URL(configured);
		} catch (_) {
			return null;
		}
	})() : null;
	const isLocalhost = (
		parsed?.hostname === "localhost" ||
		parsed?.hostname === "127.0.0.1" ||
		configured.includes("localhost") ||
		configured.includes("127.0.0.1")
	);

	// Respect explicit non-local configuration (including relative URLs).
	if (configured && (!isLocalhost)) {
		return configured;
	}

	// If we're in the browser, derive the host from the current location and keep the configured port if available.
	if (typeof window !== "undefined") {
		const {protocol, hostname} = window.location;
		const port = parsed?.port || "3000";
		const portPart = port ? `:${port}` : "";
		return `${protocol}//${hostname}${portPart}`;
	}

	// Fallback for non-browser contexts.
	return configured || "";
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
