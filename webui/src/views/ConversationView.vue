<script>
import api from "../services/api.js";
import {fileToBase64, base64ImageToDataUrl} from "../services/files.js";
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
			conversation: null,
			errormsg: null,
			loading: false,
			messageText: "",
			attachmentData: "",
			sending: false,
			refreshTimer: null,
			hoveredMessage: null,
			openActionsFor: null,
			forwardState: {
				open: false,
				loading: false,
				conversations: [],
				message: null,
				error: null,
			},
			contactInfoOpen: false,
			otherUserPhoto: "",
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
	computed: {
		otherUser() {
			if (!this.conversation?.memberDetails) return null;
			const me = sessionStore.state.user?.id;
			return this.conversation.memberDetails.find((u) => u.id !== me) || this.conversation.memberDetails[0] || null;
		},
		otherAvatarStyle() {
			const photo = base64ImageToDataUrl(this.otherUser?.photo || this.otherUserPhoto || "");
			if (photo) {
				return {
					backgroundImage: `url(${photo})`,
					backgroundSize: "cover",
					backgroundPosition: "center",
				};
			}
			return {};
		},
		otherInitial() {
			const name = this.otherUser?.name || "C";
			return name.charAt(0).toUpperCase();
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
			}
			this.errormsg = null;
			try {
				const data = await api.getConversation(this.id);
				if (data?.isGroup) {
					this.$router.replace(`/groups/${this.id}`);
					return;
				}
				this.conversation = data;
				await this.$nextTick();
				this.scrollToBottom();
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.toString();
			}
			if (!silent) {
				this.loading = false;
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
			this.openActionsFor = null;
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
			this.closeActions();
			if (!window.confirm("Eliminare il messaggio?")) return;
			this.errormsg = null;
			this.loading = true;
			try {
				await api.deleteMessage(this.id, message.id);
				await this.refresh();
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.toString();
			}
			this.loading = false;
		},
		formatTimestamp(ts) {
			if (!ts) return "";
			return new Date(ts).toLocaleString();
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
		scrollToBottom() {
			const el = this.$refs.messageThread;
			if (el && el.scrollHeight != null) {
				el.scrollTop = el.scrollHeight;
			}
		},
	},
};
</script>

<template>
  <div class="py-4">
    <div class="d-flex justify-content-between flex-wrap flex-md-nowrap align-items-center pb-2 mb-3 border-bottom">
      <div class="d-flex align-items-center gap-3">
        <div class="avatar" :style="otherAvatarStyle">
          <span v-if="!otherAvatarStyle.backgroundImage">{{ otherInitial }}</span>
        </div>
        <div>
          <h1 class="h4 mb-0">{{ conversation?.name || "Chat" }}</h1>
          <p class="text-muted small mb-0">Stile WhatsApp: Info contatto per profilo.</p>
        </div>
      </div>
      <div class="btn-toolbar gap-2">
        <button class="btn btn-outline-secondary btn-sm" @click="refresh">Aggiorna</button>
        <button v-if="otherUser" class="btn btn-outline-primary btn-sm" @click="contactInfoOpen = true">Info contatto</button>
      </div>
    </div>

    <ErrorMsg v-if="errormsg" :msg="errormsg" />

    <LoadingSpinner :loading="loading">
      <div v-if="!groupedMessages.length" class="text-muted">Nessun messaggio.</div>
      <div class="message-thread mb-4" ref="messageThread" @click="closeActions">
        <div v-for="group in groupedMessages" :key="group.key" class="message-day-group">
          <div class="date-chip">{{ group.label }}</div>
          <div
            v-for="message in group.messages"
            :key="message.id"
            class="message-row"
            :class="{'from-me': isOwnMessage(message)}"
            @mouseenter="hoveredMessage = message.id"
            @mouseleave="hoveredMessage = null"
          >
            <div class="message-bubble">
              <div class="d-flex justify-content-between align-items-center message-top">
                <div class="small text-muted" v-if="!isOwnMessage(message)">
                  {{ message.senderName || "Contatto" }}
                </div>
                <div class="small text-muted" v-else>&nbsp;</div>
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
                  >
                    {{ rx.emoji }} {{ rx.count }}
                  </span>
                </div>
                <span v-else>Nessuna reazione</span>
              </div>
              <div class="message-meta text-end text-muted small">
                {{ formatTimestamp(message.timestamp) }}
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
        </div>
      </div>

      <div class="chat-composer card shadow-sm">
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

      <div v-if="contactInfoOpen" class="forward-modal">
        <div class="forward-dialog card shadow">
          <div class="card-body text-center">
            <div class="avatar mx-auto mb-3" :style="otherAvatarStyle">
              <span v-if="!otherAvatarStyle.backgroundImage">{{ otherInitial }}</span>
            </div>
            <h5 class="mb-1">{{ otherUser?.name || "Contatto" }}</h5>
            <p class="text-muted small mb-3">Profilo contatto</p>
            <button class="btn btn-outline-secondary btn-sm" @click="contactInfoOpen = false">Chiudi</button>
          </div>
        </div>
      </div>
    </LoadingSpinner>
  </div>
</template>

<style scoped>
.chat-composer {
  position: sticky;
  bottom: 0;
  margin-top: 1rem;
}

.message-item {
  transition: background-color 0.2s ease;
}

.message-item:hover {
  background-color: #f8f9fa;
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

.message-row.from-me .message-bubble {
  background: #dcf8c6;
}

.reaction-picker {
  opacity: 0;
  transition: opacity 0.2s ease;
  margin-top: 0.25rem;
}

.reaction-picker.show {
  opacity: 1;
}

.avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background-color: #e9ecef;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  color: #6c757d;
  overflow: hidden;
}
</style>
