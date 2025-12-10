<script>
import api from "../services/api.js";

export default {
	data() {
		return {
			query: "",
			errormsg: null,
			loading: false,
			results: [],
		};
	},
	methods: {
		async search() {
			if (!this.query.trim()) {
				this.results = [];
				return;
			}
			this.loading = true;
			this.errormsg = null;
			try {
				const res = await api.searchUsers(this.query.trim());
				this.results = res.users || [];
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.toString();
			}
			this.loading = false;
		},
		async startConversation(userId) {
			if (!userId) return;
			this.errormsg = null;
			this.loading = true;
			try {
				const conv = await api.startConversation(userId);
				if (conv?.id) {
					this.$router.push(`/conversations/${conv.id}`);
				}
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
      <h1 class="h3 mb-0">Cerca utenti</h1>
    </div>

    <ErrorMsg v-if="errormsg" :msg="errormsg" />

    <div class="input-group mb-3">
      <input v-model="query" class="form-control" placeholder="username" @keyup.enter="search">
      <button class="btn btn-primary" @click="search">Cerca</button>
    </div>

    <LoadingSpinner :loading="loading">
      <div v-if="results.length === 0" class="text-muted">Nessun risultato.</div>
      <div class="list-group">
        <div v-for="user in results" :key="user.id" class="list-group-item d-flex justify-content-between align-items-center">
          <div>
            <strong>{{ user.name || "Utente" }}</strong>
          </div>
          <button class="btn btn-sm btn-outline-primary" @click="startConversation(user.id)">Apri chat</button>
        </div>
      </div>
    </LoadingSpinner>
  </div>
</template>
