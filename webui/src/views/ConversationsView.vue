<script>
import api from "../services/api.js";
import {base64ImageToDataUrl} from "../services/files.js";

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
	computed: {
		sortedConversations() {
			const list = this.conversations || [];
			return [...list].sort((a, b) => {
				const aTs = new Date(a?.lastMessage?.timestamp || 0).getTime();
				const bTs = new Date(b?.lastMessage?.timestamp || 0).getTime();
				return bTs - aTs;
			});
		},
	},
	mounted() {
		this.refresh();
		this.startAutoRefresh();
	},
	beforeUnmount() {
		this.stopAutoRefresh();
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
		async searchUsers(showAll = false) {
			const query = showAll ? "" : this.searchQuery.trim();
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
				return `${sender}: 📷 Foto`;
			}
			return `${sender}: (vuoto)`;
		},
		displayName(conv) {
			return conv?.name || "Chat";
		},
		formatTimestamp(ts) {
			if (!ts) return "";
			const parsed = new Date(ts);
			if (Number.isNaN(parsed.getTime())) return "";
			return parsed.toLocaleString();
		},
		avatarStyle(conv) {
			const photo = base64ImageToDataUrl(conv?.conversationPhoto || conv?.groupPhoto || "");
			if (photo) {
				return {
					backgroundImage: `url(${photo})`,
					backgroundSize: "cover",
					backgroundPosition: "center",
				};
			}
			return {};
		},
		avatarInitial(conv) {
			const name = this.displayName(conv) || "C";
			return name.charAt(0).toUpperCase();
		},
	},
};
</script>

<template>
  <div class="py-4">
    <div class="d-flex justify-content-between flex-wrap flex-md-nowrap align-items-center pb-2 mb-3 border-bottom">
      <div>
        <h1 class="h3 mb-0">Conversazioni</h1>
        <p class="text-muted small mb-0">Chat private e gruppi, ordinate dal messaggio più recente.</p>
      </div>
      <div class="btn-toolbar">
        <div class="input-group input-group-sm">
          <input
            v-model="searchQuery"
            class="form-control"
            placeholder="Cerca Utente"
            @keyup.enter="searchUsers"
          >
          <button class="btn btn-outline-primary" @click="searchUsers()">Cerca</button>
          <button class="btn btn-outline-secondary" @click="searchUsers(true)">Mostra tutti</button>
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
      <div v-if="sortedConversations.length === 0" class="text-muted">Nessuna conversazione.</div>
      <div v-else class="list-group">
        <RouterLink
          v-for="conv in sortedConversations"
          :key="conv.id"
          class="list-group-item list-group-item-action d-flex align-items-center gap-3"
          :to="`/conversations/${conv.id}`"
        >
          <div class="conversation-avatar" :style="avatarStyle(conv)">
            <span v-if="!avatarStyle(conv).backgroundImage">{{ avatarInitial(conv) }}</span>
          </div>
          <div class="flex-fill">
            <div class="d-flex justify-content-between align-items-center">
              <div class="d-flex align-items-center gap-2">
                <strong>{{ displayName(conv) }}</strong>
                <span v-if="conv.isGroup" class="badge bg-light text-dark border">Gruppo</span>
              </div>
              <small class="text-muted">{{ formatTimestamp(conv?.lastMessage?.timestamp) }}</small>
            </div>
            <div class="text-muted small">{{ formatPreview(conv.lastMessage) }}</div>
            <div class="small text-muted">{{ participantsLabel(conv) }}</div>
          </div>
        </RouterLink>
      </div>
    </LoadingSpinner>
  </div>
</template>

<style scoped>
.conversation-avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: #f1f3f5;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #6c757d;
  font-weight: 700;
  overflow: hidden;
  flex-shrink: 0;
}
</style>
