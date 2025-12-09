<script>
import api from "../services/api.js";
import {fileToBase64} from "../services/files.js";

export default {
	props: ["id"],
	data() {
		return {
			conversation: null,
			errormsg: null,
			loading: false,
			messageText: "",
			attachmentData: "",
			sending: false,
		};
	},
	methods: {
		async refresh() {
			this.loading = true;
			this.errormsg = null;
			try {
				this.conversation = await api.getConversation(this.id);
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.toString();
			}
			this.loading = false;
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
		async forwardMessage(message) {
			const targetConversationId = window.prompt("ID conversazione di destinazione");
			if (!targetConversationId) return;
			this.errormsg = null;
			this.loading = true;
			try {
				await api.forwardMessage(this.id, message.id, targetConversationId);
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.toString();
			}
			this.loading = false;
		},
		async deleteMessage(message) {
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
		async commentMessage(message) {
			const content = window.prompt("Aggiungi un commento");
			if (!content) return;
			this.errormsg = null;
			this.loading = true;
			try {
				await api.commentMessage(this.id, message.id, content);
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.toString();
			}
			this.loading = false;
		},
		async deleteComment(message) {
			if (!window.confirm("Rimuovere il commento?")) return;
			this.errormsg = null;
			this.loading = true;
			try {
				await api.deleteComment(this.id, message.id);
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.toString();
			}
			this.loading = false;
		},
		formatTimestamp(ts) {
			if (!ts) return "";
			return new Date(ts).toLocaleString();
		},
	},
	watch: {
		id: {
			immediate: true,
			handler() {
				this.resetComposer();
				this.refresh();
			},
		},
	},
};
</script>

<template>
	<div class="py-4">
		<div class="d-flex justify-content-between flex-wrap flex-md-nowrap align-items-center pb-2 mb-3 border-bottom">
			<div>
				<h1 class="h3 mb-0">{{ conversation?.name || "Conversazione" }}</h1>
				<p class="text-muted small mb-0">ID: {{ id }}</p>
			</div>
			<div class="btn-toolbar">
				<button class="btn btn-outline-secondary btn-sm" @click="refresh">Refresh</button>
			</div>
		</div>

		<ErrorMsg v-if="errormsg" :msg="errormsg" />

		<LoadingSpinner :loading="loading">
			<div class="card mb-3">
				<div class="card-body">
					<h5 class="card-title">Nuovo messaggio</h5>
					<div class="mb-2">
						<textarea class="form-control" rows="3" placeholder="Scrivi un messaggio" v-model="messageText"></textarea>
					</div>
					<div class="d-flex align-items-center gap-2 mb-3">
						<input ref="attachmentInput" type="file" class="form-control form-control-sm" @change="onAttachmentChange" />
						<span class="text-muted small" v-if="attachmentData">Allegato caricato</span>
					</div>
					<div class="d-flex justify-content-end">
						<button class="btn btn-primary" :disabled="sending" @click="sendMessage">Invia</button>
					</div>
				</div>
			</div>

			<div v-if="!conversation?.messages?.length" class="text-muted">Nessun messaggio.</div>
			<div class="list-group">
				<div v-for="message in conversation?.messages || []" :key="message.id" class="list-group-item">
					<div class="d-flex justify-content-between">
						<div>
							<strong>{{ message.senderName || message.senderId }}</strong>
							<span class="text-muted small ms-2">{{ formatTimestamp(message.timestamp) }}</span>
						</div>
						<div class="btn-group btn-group-sm">
							<button class="btn btn-outline-secondary" title="Inoltra" @click="forwardMessage(message)">Inoltra</button>
							<button class="btn btn-outline-secondary" title="Commenta" @click="commentMessage(message)">Commenta</button>
							<button class="btn btn-outline-secondary" title="Togli commento" @click="deleteComment(message)">Uncomment</button>
							<button class="btn btn-outline-danger" title="Elimina" @click="deleteMessage(message)">Elimina</button>
						</div>
					</div>
					<p class="mb-1">{{ message.content }}</p>
					<div class="small text-muted">
						Reazioni: {{ message.reactionCount ?? 0 }}
					</div>
				</div>
			</div>
		</LoadingSpinner>
	</div>
</template>
