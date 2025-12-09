<script>
import api from "../services/api.js";
import sessionStore from "../services/sessionStore.js";

export default {
	data() {
		return {
			conversations: [],
			errormsg: null,
			loading: false,
			newRecipientId: "",
		};
	},
	methods: {
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
		async startConversation() {
			if (!this.newRecipientId.trim()) {
				return;
			}
			this.loading = true;
			this.errormsg = null;
			try {
				const senderId = sessionStore.state.token;
				const res = await api.startConversation(senderId, this.newRecipientId.trim());
				this.newRecipientId = "";
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
	mounted() {
		this.refresh();
	},
};
</script>

<template>
	<div class="py-4">
		<div class="d-flex justify-content-between flex-wrap flex-md-nowrap align-items-center pb-2 mb-3 border-bottom">
			<h1 class="h3 mb-0">Conversazioni</h1>
			<div class="btn-toolbar">
				<div class="input-group input-group-sm">
					<input class="form-control" placeholder="ID destinatario" v-model="newRecipientId" />
					<button class="btn btn-outline-primary" @click="startConversation">Nuova chat</button>
				</div>
			</div>
		</div>

		<ErrorMsg v-if="errormsg" :msg="errormsg" />

		<LoadingSpinner :loading="loading">
			<div v-if="conversations.length === 0" class="text-muted">Nessuna conversazione.</div>
			<div class="row row-cols-1 row-cols-md-2 g-3">
				<div class="col" v-for="conv in conversations" :key="conv.id">
					<div class="card h-100">
						<div class="card-body d-flex flex-column">
							<div class="d-flex align-items-center justify-content-between">
								<h5 class="card-title mb-0">{{ conv.name || conv.id }}</h5>
								<RouterLink :to="`/conversations/${conv.id}`" class="btn btn-sm btn-primary">Apri</RouterLink>
							</div>
							<p class="text-muted small mt-2 mb-1">Partecipanti: {{ (conv.members || []).join(", ") }}</p>
							<p class="mb-0">{{ formatPreview(conv.lastMessage) }}</p>
						</div>
					</div>
				</div>
			</div>
		</LoadingSpinner>
	</div>
</template>
