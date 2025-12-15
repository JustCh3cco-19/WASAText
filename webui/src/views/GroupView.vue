<script>
import api from "../services/api.js";
import {fileToBase64, base64ImageToDataUrl} from "../services/files.js";
import {setPageTitle} from "../services/pageTitle.js";
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
			photoFileName: "",
			messageText: "",
			attachmentData: "",
			sending: false,
			refreshTimer: null,
			openActionsFor: null,
			hoveredMessage: null,
			forwardState: {
				open: false,
				loading: false,
				conversations: [],
				message: null,
				error: null,
			},
			replyingTo: null,
		};
	},
	computed: {
		groupAvatarStyle() {
			const photo = base64ImageToDataUrl(this.group?.groupPhoto || this.group?.photo || "");
			if (photo) {
				return {
					backgroundImage: `url(${photo})`,
					backgroundSize: "cover",
					backgroundPosition: "center",
				};
			}
			return {};
		},
		groupAvatarInitial() {
			const name = this.group?.name || "G";
			return name.charAt(0).toUpperCase() || "G";
		},
		groupedMessages() {
			const groups = [];
			const messages = this.conversation?.messages || [];
			for (const msg of messages) {
				const key = this.dayKey(msg.timestamp);
				const last = groups[groups.length - 1];
				if (!last || last.key !== key) {
					groups.push({
						key,
						label: this.formatDayLabel(msg.timestamp),
						messages: [],
					});
				}
				groups[groups.length - 1].messages.push(msg);
			}
			return groups;
		},
	},
	watch: {
		id: {
			immediate: true,
			handler() {
				this.resetComposer();
				this.photoFileName = "";
				this.replyingTo = null;
				setPageTitle("Gruppo");
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
				this.updatePageTitle();
				await this.$nextTick();
				if (!silent) {
					this.scrollToTop();
				}
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
			// Gestita ora nella pagina info gruppo
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
					replyTo: this.replyingTo?.id,
				});
				this.resetComposer();
				this.replyingTo = null;
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
		startReply(message) {
			this.replyingTo = message;
		},
		cancelReply() {
			this.replyingTo = null;
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
		attachmentSrc(attachment) {
			return base64ImageToDataUrl(attachment || "");
		},
		isForwarded(message) {
			return (message?.status || "").trim().toLowerCase() === "forwarded";
		},
		toggleActions(id) {
			this.openActionsFor = this.openActionsFor === id ? null : id;
		},
		closeActions() {
			this.openActionsFor = null;
		},
		dayKey(ts) {
			const date = new Date(ts);
			if (Number.isNaN(date.getTime())) return "";
			return date.toISOString().slice(0, 10);
		},
		formatDayLabel(ts) {
			const date = new Date(ts);
			if (Number.isNaN(date.getTime())) return "";
			const today = new Date();
			const todayKey = this.dayKey(today);
			const msgKey = this.dayKey(date);
			if (msgKey === todayKey) return "Oggi";
			const yesterday = new Date();
			yesterday.setDate(today.getDate() - 1);
			if (msgKey === this.dayKey(yesterday)) return "Ieri";
			const startOf = (d) => {
				const copy = new Date(d);
				copy.setHours(0, 0, 0, 0);
				return copy;
			};
			const diffDays = Math.floor((startOf(today) - startOf(date)) / (1000 * 60 * 60 * 24));
			if (diffDays >= 3) {
				return date.toLocaleDateString(undefined, {weekday: "long", day: "numeric", month: "short", year: "numeric"});
			}
			return date.toLocaleDateString(undefined, {weekday: "long", day: "numeric", month: "short"});
		},
		scrollToTop() {
			const el = this.$refs.messageThread;
			if (el && el.scrollHeight != null) {
				el.scrollTop = 0;
			}
		},
		replyPreview(message) {
			if (!message) return "";
			if (message.content) return message.content;
			if (message.attachment) return "[Allegato]";
			return "Messaggio";
		},
		deliveryBadge(message) {
			if (!this.isOwnMessage(message)) return null;
			const recipients = message.recipientCount || 0;
			if (recipients === 0) return null;
			const delivered = (message.deliveredTo || []).length;
			const read = (message.readBy || []).length;
			if (read >= recipients) {
				return {icon: "✓✓", label: `Letto da tutti (${read}/${recipients})`};
			}
			if (delivered > 0) {
				return {icon: "✓", label: `Consegnato (${delivered}/${recipients})`};
			}
			return {icon: "✓", label: "Inviato"};
		},
		reactionLabel(rx) {
			const names = rx?.userNames?.length ? rx.userNames.join(", ") : (rx?.userIds || []).join(", ");
			return names ? `${rx.emoji} ${rx.count} · ${names}` : `${rx.emoji} ${rx.count}`;
		},
		updatePageTitle() {
			const title = this.group?.name || "Gruppo";
			setPageTitle(title);
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
      <div class="btn-toolbar gap-2">
        <RouterLink :to="`/groups/${id}/info`" class="btn btn-outline-primary btn-sm">Info gruppo</RouterLink>
        <button class="btn btn-outline-danger btn-sm" @click="leaveGroup">Esci</button>
      </div>
    </div>

    <ErrorMsg v-if="errormsg" :msg="errormsg" />

    <div class="row">
      <div class="col-12">
        <LoadingSpinner :loading="loadingChat || loading">
          <div v-if="!groupedMessages.length" class="text-muted">Nessun messaggio.</div>
          <div ref="messageThread" class="message-thread mb-4" @click="closeActions">
            <div v-for="msgGroup in groupedMessages" :key="msgGroup.key" class="message-day-group">
              <div class="date-chip">{{ msgGroup.label }}</div>
              <div
                v-for="message in msgGroup.messages"
                :key="message.id"
                class="message-row"
                :class="{'from-me': isOwnMessage(message)}"
              >
                <div class="message-bubble">
                  <div class="d-flex justify-content-between align-items-center message-top">
                    <div class="small text-muted">
                      <span v-if="!isOwnMessage(message)">{{ message.senderName || "Membro" }}</span>
                      <span v-else>&nbsp;</span>
                    </div>
                    <div class="message-actions">
                      <button
                        type="button"
                        class="btn btn-sm btn-outline-secondary"
                        title="Azioni"
                        @click.stop="toggleActions(message.id)"
                      >
                        ...
                      </button>
                      <div v-if="openActionsFor === message.id" class="actions-menu card shadow-sm">
                        <button type="button" class="dropdown-item" @click.stop="startReply(message)">Rispondi</button>
                        <button type="button" class="dropdown-item" @click.stop="openForwardModal(message)">Inoltra</button>
                        <button
                          v-if="isOwnMessage(message)"
                          type="button"
                          class="dropdown-item text-danger"
                          @click.stop="deleteMessage(message)"
                        >
                          Elimina
                        </button>
                      </div>
                    </div>
                  </div>
                  <div v-if="isForwarded(message)" class="forwarded-label">
                    <span class="arrow">↪</span>
                    <em>Messaggio inoltrato</em>
                  </div>
                  <div v-if="message.replyTo" class="reply-chip">
                    <div class="small text-muted">Risposta a {{ message.replySenderName || "messaggio" }}</div>
                    <div class="reply-text">
                      {{ message.replyContent || (message.replyAttachment ? "[Allegato]" : "Messaggio") }}
                    </div>
                  </div>
                  <div class="mb-1">
                    <p v-if="message.content" class="mb-1">{{ message.content }}</p>
                    <p v-else-if="message.attachment" class="mb-1 text-muted small">[Allegato]</p>
                    <div v-if="message.attachment" class="mt-1">
                      <img
                        :src="attachmentSrc(message.attachment)"
                        alt="Allegato"
                        class="message-attachment rounded border"
                      >
                    </div>
                  </div>
                  <div class="small text-muted d-flex align-items-center gap-2 flex-wrap">
                    <div v-if="message.reactions?.length" class="d-flex gap-2 flex-wrap">
                      <span
                        v-for="rx in message.reactions"
                        :key="rx.emoji"
                        class="badge bg-light text-dark border"
                        :title="reactionLabel(rx)"
                      >
                        {{ rx.emoji }} {{ rx.count }}
                        <span v-if="rx.userNames?.length" class="ms-1 text-muted">({{ rx.userNames.join(', ') }})</span>
                      </span>
                    </div>
                    <span v-else>Nessuna reazione</span>
                  </div>
                  <div class="message-meta text-muted small">
                    <span class="timestamp">{{ formatTimestamp(message.timestamp) }}</span>
                    <span
                      v-if="deliveryBadge(message)"
                      class="delivery-icon"
                      :title="deliveryBadge(message).label"
                    >
                      {{ deliveryBadge(message).icon }}
                    </span>
                  </div>
                  <div class="reaction-picker">
                    <button
                      v-for="emoji in emojiOptions()"
                      :key="emoji"
                      class="btn btn-light btn-sm"
                      :class="{'border-primary': myReaction(message) === emoji}"
                      :title="`Reagisci con ${emoji}`"
                      @click="toggleReaction(message, emoji)"
                    >
                      {{ emoji }}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class="chat-composer card shadow-sm">
            <div class="card-body">
              <div v-if="replyingTo" class="alert alert-secondary py-2 px-3 d-flex justify-content-between align-items-center">
                <div>
                  <div class="small text-muted">Rispondi a {{ replyingTo.senderName || "messaggio" }}</div>
                  <div class="fw-semibold">{{ replyPreview(replyingTo) }}</div>
                </div>
                <button class="btn btn-sm btn-outline-secondary" @click="cancelReply">Annulla</button>
              </div>
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
    </div>
  </div>
</template>

<style scoped>
.chat-composer {
  position: sticky;
  bottom: 0;
  margin-top: 0.5rem;
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

.message-thread {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.message-day-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.date-chip {
  position: sticky;
  top: 0;
  align-self: center;
  z-index: 2;
  padding: 4px 12px;
  background: #e9ecef;
  color: #495057;
  border-radius: 999px;
  font-weight: 600;
  text-transform: capitalize;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.08);
}

.message-row {
  display: flex;
  justify-content: flex-start;
}

.message-row.from-me {
  justify-content: flex-end;
}

.message-bubble {
  background: #f1f3f5;
  padding: 0.5rem 0.75rem;
  border-radius: 12px;
  max-width: 80%;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.08);
}

.message-row.from-me .message-bubble {
  background: #dcf8c6;
}

.message-attachment {
  max-width: 220px;
  max-height: 220px;
  object-fit: cover;
  display: block;
}

.message-top {
  gap: 0.5rem;
}

.message-actions {
  position: relative;
  z-index: 6;
}

.actions-menu {
  position: absolute;
  right: 0;
  top: 100%;
  min-width: 140px;
  z-index: 7;
}

.message-meta {
  margin-top: 0.25rem;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
  color: #6c757d;
}

.message-meta .timestamp {
  line-height: 1;
}

.message-meta .delivery-icon {
  font-size: 0.9em;
  line-height: 1;
  margin-left: 2px;
}

.forwarded-label {
  color: #6c757d;
  font-style: italic;
  display: flex;
  align-items: center;
  gap: 0.35rem;
  margin-top: 0.25rem;
}

.forwarded-label .arrow {
  font-size: 14px;
  line-height: 1;
}

.reply-chip {
  background: rgba(0, 0, 0, 0.04);
  border-left: 3px solid #6c757d;
  padding: 4px 8px;
  border-radius: 8px;
  margin-bottom: 6px;
}

.reaction-picker {
  opacity: 1;
  margin-top: 0.25rem;
}
</style>
