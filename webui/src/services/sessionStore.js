import {reactive} from "vue";

let storedUser = null;
try {
	const raw = localStorage.getItem("user");
	if (raw) {
		storedUser = JSON.parse(raw);
	}
} catch (e) {
	storedUser = null;
}

const state = reactive({
	token: localStorage.getItem("token") || null,
	user: storedUser,
});

function setToken(token) {
	state.token = token;
	if (token) {
		localStorage.setItem("token", token);
	} else {
		localStorage.removeItem("token");
	}
}

function setUser(user) {
	state.user = user || null;
	if (user) {
		localStorage.setItem("user", JSON.stringify(user));
	} else {
		localStorage.removeItem("user");
	}
}

function logout() {
	setToken(null);
	setUser(null);
}

function isAuthenticated() {
	return !!state.token;
}

export default {
	state,
	setToken,
	setUser,
	logout,
	isAuthenticated,
};
