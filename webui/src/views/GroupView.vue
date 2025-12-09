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
			newUserId: "",
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
		async addUser() {
			if (!this.newUserId.trim()) return;
			this.addingUser = true;
			this.errormsg = null;
			try {
				await api.addToGroup(this.id, this.newUserId.trim());
				this.newUserId = "";
				await this.refresh();
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.toString();
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
								<input class="form-control" placeholder="User ID" v-model="newUserId" />
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
						<div v-for="member in group?.members || []" :key="member" class="list-group-item">
							{{ member }}
						</div>
						<div v-if="!group?.members?.length" class="text-muted">Nessun membro.</div>
					</div>
				</div>
			</div>
		</LoadingSpinner>
	</div>
</template>
