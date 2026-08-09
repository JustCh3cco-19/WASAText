<script>
import api from "../services/api.js";
import sessionStore from "../services/sessionStore.js";

export default {
	data() {
		return {
			name: "",
			password: "",
			registering: false,
			errormsg: null,
			loading: false,
		};
	},
	mounted() {
		if (sessionStore.isAuthenticated()) {
			this.$router.push("/conversations");
		}
	},
	methods: {
		async submit() {
			if (!this.name.trim() || this.password.length < 10) {
				this.errormsg = "Inserisci username e password (almeno 10 caratteri).";
				return;
			}
			this.loading = true;
			this.errormsg = null;
			try {
				const res = this.registering
					? await api.register(this.name.trim(), this.password)
					: await api.login(this.name.trim(), this.password);
				if (res.user) {
					sessionStore.setUser(res.user);
				} else {
					sessionStore.setUser({id: this.name.trim(), name: this.name.trim()});
				}
				const redirect = this.$route.query.redirect || "/conversations";
				this.$router.push(redirect);
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.toString();
			}
			this.loading = false;
		},
	},
};
</script>

<template>
  <div class="py-4">
    <div class="d-flex justify-content-between flex-wrap flex-md-nowrap align-items-center pb-2 mb-3 border-bottom">
      <h1 class="h3">{{ registering ? "Registrazione" : "Login" }}</h1>
    </div>
    <ErrorMsg v-if="errormsg" :msg="errormsg" />
    <LoadingSpinner :loading="loading">
      <form class="card p-3" @submit.prevent="submit">
        <div class="mb-3">
          <label class="form-label">Nome utente</label>
          <input v-model="name" class="form-control" placeholder="es. justchecco" required minlength="3" maxlength="16">
        </div>
        <div class="mb-3">
          <label class="form-label">Password</label>
          <input v-model="password" class="form-control" type="password" autocomplete="current-password" required minlength="10" maxlength="128">
          <div class="form-text">Usa almeno 10 caratteri.</div>
        </div>
        <div class="d-flex justify-content-between align-items-center">
          <button type="button" class="btn btn-link px-0" @click="registering = !registering">
            {{ registering ? "Hai già un account?" : "Crea un account" }}
          </button>
          <button type="submit" class="btn btn-primary">{{ registering ? "Registrati" : "Entra" }}</button>
        </div>
      </form>
    </LoadingSpinner>
  </div>
</template>
