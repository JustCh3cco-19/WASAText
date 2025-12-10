<script>
import api from "../services/api.js";
import {fileToBase64} from "../services/files.js";
import sessionStore from "../services/sessionStore.js";

const DEFAULT_REACTIONS = ["👍", "❤️", "😂", "😮", "🎉"];

export default {
	props: {
		id: {
			type: String,
			required: true,
		},
	},
	data() {
		return {
			group: null,
			conversation: null,
			errormsg: null,
			loading: false,
			loadingChat: false,
			updatingName: false,
			newName: "",
			addingUser: false,
			newUserQuery: "",
			updatingPhoto: false,
			messageText: "",
			attachmentData: "",
			sending: false,
			refreshTimer: null,
			hoveredMessage: null,
			forwardState: {
				open: false,
				loading: false,
				conversations: [],
				message: null,
				error: null,
			},
		};
	},
	watch: {
		id: {
			immediate: true,
			handler() {
				this.resetComposer();
				this.refresh();
				this.startAutoRefresh();
			},
		},
	},
	beforeUnmount() {
		this.stopAutoRefresh();
	},
	methods: {
		startAutoRefresh() {
			this.stopAutoRefresh();
			this.refreshTimer = setInterval(() => {
				this.refresh(true);
			}, 4000);
		},
		stopAutoRefresh() {
			if (this.refreshTimer) {
				clearInterval(this.refreshTimer);
				this.refreshTimer = null;
			}
		},
		async refresh() {
			const silent = arguments.length && arguments[0] === true;
			if (!silent) {
				this.loading = true;
				this.loadingChat = true;
			}
			this.errormsg = null;
			try {
				const [groupRes, convRes] = await Promise.all([
					api.getGroup(this.id),
					api.getConversation(this.id),
				]);
				this.group = groupRes;
				this.newName = this.group?.name || "";
				this.conversation = convRes;
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.toString();
			}
			if (!silent) {
				this.loading = false;
				this.loadingChat = false;
			}
		},
		resetComposer() {
			this.messageText = "";
			this.attachmentData = "";
			const input = this.$refs.attachmentInput;
			if (input) {
				input.value = null;
			}
		},
		async updateName() {
			if (!this.newName.trim()) return;
			this.updatingName = true;
			this.errormsg = null;
			try {
				await api.updateGroupName(this.id, this.newName.trim());
				await this.refresh();
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.toString();
			}
			this.updatingName = false;
		},
		async onPhotoChange(event) {
			const file = event?.target?.files?.[0];
			if (!file) return;
			this.updatingPhoto = true;
			this.errormsg = null;
			try {
				const base64 = await fileToBase64(file);
				await api.updateGroupPhoto(this.id, base64, file.type || "image/png");
				await this.refresh();
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.toString();
			}
			this.updatingPhoto = false;
		},
		async resolveUserId(query) {
			const trimmed = (query || "").trim();
			if (!trimmed) {
				throw new Error("Inserisci un nome utente");
			}
			// Try to resolve by name (preferred). If nothing matches and looks like an ID, fall back.
			try {
				const res = await api.searchUsers(trimmed);
				const users = res.users || [];
				const exact = users.find(
					(u) =>
						(u.name && u.name.toLowerCase() === trimmed.toLowerCase()) ||
						u.id === trimmed
				);
				if (exact) {
					return exact.id;
				}
				if (users.length === 1) {
					return users[0].id;
				}
			} catch (err) {
				// ignore and try fallback
			}
			if (/^[a-zA-Z0-9_-]{8,}$/.test(trimmed)) {
				return trimmed;
			}
			throw new Error(`Utente "${trimmed}" non trovato`);
		},
		async addUser() {
			if (!this.newUserQuery.trim()) return;
			this.addingUser = true;
			this.errormsg = null;
			try {
				const userId = await this.resolveUserId(this.newUserQuery);
				await api.addToGroup(this.id, userId);
				this.newUserQuery = "";
				await this.refresh();
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.message || e.toString();
			}
			this.addingUser = false;
		},
		async leaveGroup() {
			if (!window.confirm("Vuoi uscire dal gruppo?")) return;
			this.errormsg = null;
			this.loading = true;
			try {
				await api.leaveGroup(this.id);
				this.$router.push("/groups");
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.toString();
			}
			this.loading = false;
		},
		memberList() {
			if (this.group?.memberDetails?.length) {
				return this.group.memberDetails;
			}
			const count = (this.group?.members || []).length;
			return Array.from({length: count}, (_, idx) => ({name: `Membro ${idx + 1}`}));
		},
		async sendMessage() {
			if (!this.messageText.trim() && !this.attachmentData) {
				return;
			}
			this.sending = true;
			this.errormsg = null;
			try {
				await api.sendMessage(this.id, {
					content: this.messageText.trim(),
					attachment: this.attachmentData,
				});
				this.resetComposer();
				await this.refresh();
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.toString();
			}
			this.sending = false;
		},
		async onAttachmentChange(event) {
			const file = event?.target?.files?.[0];
			this.attachmentData = file ? await fileToBase64(file) : "";
		},
		handleKeydown(e) {
			if (e.key === "Enter" && e.ctrlKey) {
				e.preventDefault();
				this.sendMessage();
			}
		},
		async openForwardModal(message) {
			this.forwardState.open = true;
			this.forwardState.message = message;
			this.forwardState.loading = true;
			this.forwardState.error = null;
			try {
				const res = await api.getConversations();
				this.forwardState.conversations = res.conversations || [];
			} catch (e) {
				this.forwardState.error = e?.response?.data?.error || e.toString();
			}
			this.forwardState.loading = false;
		},
		closeForwardModal() {
			this.forwardState.open = false;
			this.forwardState.message = null;
			this.forwardState.error = null;
		},
		async forwardToConversation(targetConversationId) {
			if (!targetConversationId || !this.forwardState.message) return;
			this.errormsg = null;
			this.forwardState.loading = true;
			try {
				await api.forwardMessage(this.id, this.forwardState.message.id, targetConversationId);
				this.closeForwardModal();
			} catch (e) {
				this.forwardState.error = e?.response?.data?.error || e.toString();
			}
			this.forwardState.loading = false;
		},
		async deleteMessage(message) {
			if (!window.confirm("Eliminare il messaggio?")) return;
			this.errormsg = null;
			this.loadingChat = true;
			try {
				await api.deleteMessage(this.id, message.id);
				await this.refresh();
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.toString();
			}
			this.loadingChat = false;
		},
		formatTimestamp(ts) {
			if (!ts) return "";
			return new Date(ts).toLocaleString();
		},
		isOwnMessage(message) {
			const currentUserId = sessionStore.state.user?.id;
			return !!currentUserId && message?.senderId === currentUserId;
		},
		myReaction(message) {
			const me = sessionStore.state.user?.id;
			if (!me || !message?.reactions) return null;
			for (const rx of message.reactions) {
				if ((rx.userIds || []).includes(me)) {
					return rx.emoji || "👍";
				}
			}
			return null;
		},
		async toggleReaction(message, emoji) {
			const current = this.myReaction(message);
			this.errormsg = null;
			try {
				if (current === emoji) {
					await api.deleteComment(this.id, message.id);
				} else {
					await api.commentMessage(this.id, message.id, emoji);
				}
				await this.refresh(true);
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.toString();
			}
		},
		emojiOptions() {
			return DEFAULT_REACTIONS;
		},
	},
};
</script>

<template>
  <div class="py-4">
            <div class="d-flex justify-content-between flex-wrap flex-md-nowrap align-items-center pb-2 mb-3 border-bottom">
              <div>
                <h1 class="h3 mb-0">{{ group?.name || "Gruppo" }}</h1>
              </div>
              <div class="btn-toolbar">
        <button class="btn btn-outline-secondary btn-sm me-2" @click="refresh">Refresh</button>
        <button class="btn btn-outline-danger btn-sm" @click="leaveGroup">Esci</button>
      </div>
    </div>

    <ErrorMsg v-if="errormsg" :msg="errormsg" />

    <div class="row">
      <div class="col-lg-7">
        <LoadingSpinner :loading="loadingChat || loading">
          <div class="card mb-3">
            <div class="card-body">
              <h5 class="card-title">Chat di gruppo</h5>
              <div class="chat-composer card shadow-sm border-0">
                <div class="card-body">
                  <div class="d-flex align-items-center gap-2">
                    <button class="btn btn-light d-flex align-items-center" type="button" title="Allega" @click="$refs.attachmentInput.click()">
                      <svg class="feather"><use href="/feather-sprite-v4.29.0.svg#paperclip" /></svg>
                    </button>
                    <input ref="attachmentInput" type="file" class="d-none" @change="onAttachmentChange">
                    <textarea
                      v-model="messageText"
                      class="form-control flex-fill"
                      rows="2"
                      placeholder="Scrivi un messaggio"
                      @keydown="handleKeydown"
                    />
                    <button class="btn btn-primary" :disabled="sending" @click="sendMessage">Invia</button>
                  </div>
                  <div class="text-muted small mt-1">
                    <span v-if="attachmentData">Allegato pronto per l'invio. </span>
                    Invio con Ctrl+Invio, Invio va a capo.
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div v-if="!conversation?.messages?.length" class="text-muted">Nessun messaggio.</div>
          <div class="list-group mb-4">
            <div
              v-for="message in conversation?.messages || []"
              :key="message.id"
              class="list-group-item message-item position-relative"
              @mouseenter="hoveredMessage = message.id"
              @mouseleave="hoveredMessage = null"
            >
              <div class="d-flex justify-content-between">
                <div>
                  <strong>{{ message.senderName || message.senderId }}</strong>
                  <span class="text-muted small ms-2">{{ formatTimestamp(message.timestamp) }}</span>
                </div>
                <div class="btn-group btn-group-sm">
                  <button class="btn btn-outline-secondary" title="Inoltra" @click="openForwardModal(message)">Inoltra</button>
                  <button
                    v-if="isOwnMessage(message)"
                    class="btn btn-outline-danger"
                    title="Elimina"
                    @click="deleteMessage(message)"
                  >
                    Elimina
                  </button>
                </div>
              </div>
              <p class="mb-1">{{ message.content }}</p>
              <div class="small text-muted d-flex align-items-center gap-2 flex-wrap">
                <div v-if="message.reactions?.length" class="d-flex gap-2 flex-wrap">
                  <span
                    v-for="rx in message.reactions"
                    :key="rx.emoji"
                    class="badge bg-light text-dark border"
                  >
                    {{ rx.emoji }} {{ rx.count }}
                  </span>
                </div>
                <span v-else>Nessuna reazione</span>
              </div>
              <div class="reaction-picker" :class="{'show': hoveredMessage === message.id}">
                <button
                  v-for="emoji in emojiOptions()"
                  :key="emoji"
                  class="btn btn-light btn-sm"
                  :class="{'border-primary': myReaction(message) === emoji}"
                  @click="toggleReaction(message, emoji)"
                  :title="`Reagisci con ${emoji}`"
                >
                  {{ emoji }}
                </button>
              </div>
            </div>
          </div>

          <div v-if="forwardState.open" class="forward-modal">
            <div class="forward-dialog card shadow">
              <div class="card-body">
                <div class="d-flex justify-content-between align-items-center mb-2">
                  <h5 class="card-title mb-0">Inoltra messaggio</h5>
                  <button class="btn btn-sm btn-outline-secondary" @click="closeForwardModal">Chiudi</button>
                </div>
                <p class="text-muted small mb-2">Seleziona una chat o un gruppo</p>
                <div v-if="forwardState.error" class="text-danger small mb-2">{{ forwardState.error }}</div>
                <div v-if="forwardState.loading" class="text-muted">Caricamento…</div>
                <div v-else class="list-group forward-list">
                  <button
                    v-for="conv in forwardState.conversations"
                    :key="conv.id"
                    class="list-group-item list-group-item-action d-flex justify-content-between align-items-center"
                    @click="forwardToConversation(conv.id)"
                  >
                    <span>{{ conv.name || conv.id }}</span>
                    <span class="badge bg-light text-dark">{{ conv.isGroup ? "Gruppo" : "Chat" }}</span>
                  </button>
                  <div v-if="forwardState.conversations.length === 0" class="text-muted small">
                    Nessuna chat disponibile.
                  </div>
                </div>
              </div>
            </div>
          </div>
        </LoadingSpinner>
      </div>

      <div class="col-lg-5">
        <LoadingSpinner :loading="loading">
          <div class="card mb-3">
            <div class="card-body">
              <h5 class="card-title">Impostazioni gruppo</h5>
              <div class="row g-2">
                <div class="col-md-12">
                  <label class="form-label">Nome</label>
                  <div class="input-group">
                    <input v-model="newName" class="form-control">
                    <button class="btn btn-primary" :disabled="updatingName" @click="updateName">Salva</button>
                  </div>
                </div>
                <div class="col-md-12">
                  <label class="form-label">Foto</label>
                  <input class="form-control" type="file" :disabled="updatingPhoto" @change="onPhotoChange">
                </div>
              </div>
              <div class="mt-3 row g-2">
                <div class="col-md-12">
                  <label class="form-label">Aggiungi utente</label>
                  <div class="input-group">
                    <input
                      v-model="newUserQuery"
                      class="form-control"
                      placeholder="Nome utente"
                      @keyup.enter="addUser"
                    >
                    <button class="btn btn-secondary" :disabled="addingUser" @click="addUser">Aggiungi</button>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class="card">
            <div class="card-body">
              <h5 class="card-title">Membri</h5>
              <div class="list-group">
                <div v-for="member in memberList()" :key="member.id || member" class="list-group-item">
                  <strong>{{ member.name || member.id }}</strong>
                </div>
                <div v-if="memberList().length === 0" class="text-muted">Nessun membro.</div>
              </div>
            </div>
          </div>
        </LoadingSpinner>
      </div>
    </div>
  </div>
</template>

<style scoped>
.chat-composer {
  position: sticky;
  bottom: 0;
  margin-top: 0.5rem;
}

.message-item {
  transition: background-color 0.2s ease;
}

.message-item:hover {
  background-color: #f8f9fa;
}

.reaction-picker {
  opacity: 0;
  transition: opacity 0.2s ease;
  margin-top: 0.25rem;
}

.reaction-picker.show {
  opacity: 1;
}

.forward-modal {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.35);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1050;
}

.forward-dialog {
  width: min(520px, 95vw);
}

.forward-list {
  max-height: 260px;
  overflow-y: auto;
}
</style>
