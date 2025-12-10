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
			refreshTimer: null,
		};
	},
	mounted() {
		this.refresh();
		this.startAutoRefresh();
	},
	beforeUnmount() {
		this.stopAutoRefresh();
	},
	computed: {
		privateChats() {
			return (this.conversations || []).filter((c) => !c.isGroup);
		},
	},
	methods: {
		startAutoRefresh() {
			this.stopAutoRefresh();
			this.refreshTimer = setInterval(() => {
				this.refresh(true);
			}, 5000);
		},
		stopAutoRefresh() {
			if (this.refreshTimer) {
				clearInterval(this.refreshTimer);
				this.refreshTimer = null;
			}
		},
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
		async refresh(silent = false) {
			if (!silent) {
				this.loading = true;
			}
			this.errormsg = null;
			try {
				const res = await api.getConversations();
				this.conversations = res.conversations || [];
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.toString();
			}
			if (!silent) {
				this.loading = false;
			}
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
			const sender = message.senderName || message.senderId || "?";
			const content = (message.content || "").trim();
			const hasAttachment = !!message.attachment;
			if (content && hasAttachment) {
				return `${sender}: ${content} 📎`;
			}
			if (content) {
				return `${sender}: ${content}`;
			}
			if (hasAttachment) {
				return `${sender}: [Allegato]`;
			}
			return `${sender}: (vuoto)`;
		},
		displayName(conv) {
			return conv?.name || "Chat";
		},
	},
};
</script>

<template>
  <div class="py-4">
    <div class="d-flex justify-content-between flex-wrap flex-md-nowrap align-items-center pb-2 mb-3 border-bottom">
      <div>
        <h1 class="h3 mb-0">Chat private</h1>
        <p class="text-muted mb-0">Solo chat 1:1. Per i gruppi usa la sezione “Gruppi”.</p>
      </div>
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
          </div>
          <button class="btn btn-sm btn-outline-primary" @click="startConversation(user.id)">Apri chat</button>
        </div>
      </div>
    </LoadingSpinner>

    <LoadingSpinner :loading="loading">
      <div v-if="privateChats.length === 0" class="text-muted">Nessuna chat privata.</div>
      <div class="row row-cols-1 row-cols-md-2 g-3">
        <div v-for="conv in privateChats" :key="conv.id" class="col">
          <div class="card h-100">
            <div class="card-body d-flex flex-column">
              <div class="d-flex align-items-center justify-content-between">
                <h5 class="card-title mb-0">{{ displayName(conv) }}</h5>
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
