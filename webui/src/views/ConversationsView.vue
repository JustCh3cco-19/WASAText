<script>
import api from "../services/api.js";

export default {
	data() {
		return {
			conversations: [],
			errormsg: null,
			loading: false,
			searchQuery: "",
			searchResults: [],
			searching: false,
			hasSearched: false,
		};
	},
	mounted() {
		this.refresh();
	},
	methods: {
		participantsLabel(conv) {
			const details = conv?.memberDetails || [];
			const names = details.map((u) => u?.name).filter(Boolean);
			if (names.length) {
				return names.join(", ");
			}
			if (details.length) {
				return `${details.length} membri`;
			}
			const memberCount = (conv?.members || []).length;
			if (memberCount) {
				return `${memberCount} membri`;
			}
			return "Partecipanti non disponibili";
		},
		async refresh() {
			this.loading = true;
			this.errormsg = null;
			try {
				const res = await api.getConversations();
				this.conversations = res.conversations || [];
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.toString();
			}
			this.loading = false;
		},
		async searchUsers() {
			const query = this.searchQuery.trim();
			if (!query) {
				this.searchResults = [];
				this.hasSearched = false;
				return;
			}
			this.searching = true;
			this.errormsg = null;
			try {
				const res = await api.searchUsers(query);
				this.searchResults = res.users || [];
				this.hasSearched = true;
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.toString();
			}
			this.searching = false;
		},
		async startConversation(recipientId) {
			if (!recipientId) {
				return;
			}
			this.loading = true;
			this.errormsg = null;
			try {
				const res = await api.startConversation(recipientId);
				await this.refresh();
				if (res?.id) {
					this.$router.push(`/conversations/${res.id}`);
				}
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.toString();
			}
			this.loading = false;
		},
		formatPreview(message) {
			if (!message) return "Nessun messaggio";
			return `${message.senderName || message.senderId || "?"}: ${message.content || ""}`;
		},
	},
};
</script>

<template>
  <div class="py-4">
    <div class="d-flex justify-content-between flex-wrap flex-md-nowrap align-items-center pb-2 mb-3 border-bottom">
      <h1 class="h3 mb-0">Conversazioni</h1>
      <div class="btn-toolbar">
        <div class="input-group input-group-sm">
          <input
            v-model="searchQuery"
            class="form-control"
            placeholder="Cerca per nome utente"
            @keyup.enter="searchUsers"
          >
          <button class="btn btn-outline-primary" @click="searchUsers">Cerca</button>
        </div>
      </div>
    </div>

    <ErrorMsg v-if="errormsg" :msg="errormsg" />

    <LoadingSpinner :loading="searching">
      <div v-if="searchResults.length === 0 && hasSearched && !searching" class="text-muted mb-3">
        Nessun utente trovato.
      </div>
      <div v-if="searchResults.length" class="list-group mb-3">
        <div
          v-for="user in searchResults"
          :key="user.id"
          class="list-group-item d-flex justify-content-between align-items-center"
        >
          <div>
            <strong>{{ user.name || user.id }}</strong>
            <span class="text-muted small ms-2">{{ user.id }}</span>
          </div>
          <button class="btn btn-sm btn-outline-primary" @click="startConversation(user.id)">Apri chat</button>
        </div>
      </div>
    </LoadingSpinner>

    <LoadingSpinner :loading="loading">
      <div v-if="conversations.length === 0" class="text-muted">Nessuna conversazione.</div>
      <div class="row row-cols-1 row-cols-md-2 g-3">
        <div v-for="conv in conversations" :key="conv.id" class="col">
          <div class="card h-100">
            <div class="card-body d-flex flex-column">
              <div class="d-flex align-items-center justify-content-between">
                <h5 class="card-title mb-0">{{ conv.name || conv.id }}</h5>
                <RouterLink :to="`/conversations/${conv.id}`" class="btn btn-sm btn-primary">Apri</RouterLink>
              </div>
              <p class="text-muted small mt-2 mb-1">Partecipanti: {{ participantsLabel(conv) }}</p>
              <p class="mb-0">{{ formatPreview(conv.lastMessage) }}</p>
            </div>
          </div>
        </div>
      </div>
    </LoadingSpinner>
  </div>
</template>
