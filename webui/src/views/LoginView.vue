<script>
import api from "../services/api.js";
import sessionStore from "../services/sessionStore.js";

export default {
	data() {
		return {
			name: "",
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
			if (!this.name.trim()) {
				this.errormsg = "Inserisci un nome valido.";
				return;
			}
			this.loading = true;
			this.errormsg = null;
			try {
				const res = await api.login(this.name.trim());
				const token = res.identifier || res.token || res.id || this.name.trim();
				sessionStore.setToken(token);
				sessionStore.setUser({id: token, name: this.name.trim()});
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
      <h1 class="h3">Login</h1>
    </div>
    <ErrorMsg v-if="errormsg" :msg="errormsg" />
    <LoadingSpinner :loading="loading">
      <form class="card p-3" @submit.prevent="submit">
        <div class="mb-3">
          <label class="form-label">Nome utente</label>
          <input v-model="name" class="form-control" placeholder="es. justchecco" required minlength="3" maxlength="16">
        </div>
        <div class="d-flex justify-content-end">
          <button type="submit" class="btn btn-primary">Entra</button>
        </div>
      </form>
    </LoadingSpinner>
  </div>
</template>
