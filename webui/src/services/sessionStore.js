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
	user: storedUser,
});

// Remove bearer tokens persisted by releases prior to HttpOnly cookie sessions.
localStorage.removeItem("token");

function setUser(user) {
	state.user = user || null;
	if (user) {
		localStorage.setItem("user", JSON.stringify(user));
	} else {
		localStorage.removeItem("user");
	}
}

function logout() {
	setUser(null);
	localStorage.removeItem("token");
}

function isAuthenticated() {
	return !!state.user;
}

export default {
	state,
	setUser,
	logout,
	isAuthenticated,
};
