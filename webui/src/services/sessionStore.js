import {reactive} from "vue";

const state = reactive({
	token: localStorage.getItem("token") || null,
	user: null,
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
	state.user = user;
}

function logout() {
	setToken(null);
	state.user = null;
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
