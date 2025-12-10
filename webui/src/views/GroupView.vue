<script>
import api from "../services/api.js";
import {fileToBase64} from "../services/files.js";

export default {
	props: ["id"],
	data() {
		return {
			group: null,
			errormsg: null,
			loading: false,
			updatingName: false,
			newName: "",
			addingUser: false,
			newUserQuery: "",
			updatingPhoto: false,
		};
	},
	methods: {
		async refresh() {
			this.loading = true;
			this.errormsg = null;
			try {
				this.group = await api.getGroup(this.id);
				this.newName = this.group?.name || "";
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.toString();
			}
			this.loading = false;
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
	},
	watch: {
		id: {
			immediate: true,
			handler() {
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
				<h1 class="h3 mb-0">{{ group?.name || "Gruppo" }}</h1>
				<p class="text-muted small mb-0">ID: {{ id }}</p>
			</div>
			<div class="btn-toolbar">
				<button class="btn btn-outline-secondary btn-sm me-2" @click="refresh">Refresh</button>
				<button class="btn btn-outline-danger btn-sm" @click="leaveGroup">Esci</button>
			</div>
		</div>

		<ErrorMsg v-if="errormsg" :msg="errormsg" />

		<LoadingSpinner :loading="loading">
			<div class="card mb-3">
				<div class="card-body">
					<h5 class="card-title">Impostazioni gruppo</h5>
					<div class="row g-2">
						<div class="col-md-6">
							<label class="form-label">Nome</label>
							<div class="input-group">
								<input class="form-control" v-model="newName" />
								<button class="btn btn-primary" :disabled="updatingName" @click="updateName">Salva</button>
							</div>
						</div>
						<div class="col-md-6">
							<label class="form-label">Foto</label>
							<input class="form-control" type="file" @change="onPhotoChange" :disabled="updatingPhoto" />
						</div>
					</div>
					<div class="mt-3 row g-2">
						<div class="col-md-6">
							<label class="form-label">Aggiungi utente</label>
							<div class="input-group">
								<input
									class="form-control"
									placeholder="Nome utente"
									v-model="newUserQuery"
									@keyup.enter="addUser"
								/>
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
</template>
