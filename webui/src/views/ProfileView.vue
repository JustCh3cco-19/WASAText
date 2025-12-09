<script>
import api from "../services/api.js";
import sessionStore from "../services/sessionStore.js";
import {fileToBase64} from "../services/files.js";

export default {
	data() {
		return {
			errormsg: null,
			loading: false,
			newName: sessionStore.state.user?.name || "",
			newPhoto: "",
			updatingName: false,
			updatingPhoto: false,
		};
	},
	methods: {
		async updateName() {
			if (!this.newName.trim()) return;
			this.updatingName = true;
			this.errormsg = null;
			try {
				const user = await api.setMyName(this.newName.trim());
				sessionStore.setUser(user);
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.toString();
			}
			this.updatingName = false;
		},
		async onPhotoChange(event) {
			const file = event?.target?.files?.[0];
			this.newPhoto = file ? await fileToBase64(file) : "";
		},
		async updatePhoto() {
			if (!this.newPhoto) return;
			this.updatingPhoto = true;
			this.errormsg = null;
			try {
				const user = await api.setMyPhoto(this.newPhoto);
				sessionStore.setUser(user);
				this.newPhoto = "";
				const input = this.$refs.photoInput;
				if (input) input.value = null;
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.toString();
			}
			this.updatingPhoto = false;
		},
	},
};
</script>

<template>
	<div class="py-4">
		<div class="d-flex justify-content-between flex-wrap flex-md-nowrap align-items-center pb-2 mb-3 border-bottom">
			<h1 class="h3 mb-0">Profilo</h1>
		</div>

		<ErrorMsg v-if="errormsg" :msg="errormsg" />

		<div class="card">
			<div class="card-body">
				<h5 class="card-title">Impostazioni</h5>
				<div class="mb-3">
					<label class="form-label">Nome</label>
					<div class="input-group">
						<input class="form-control" v-model="newName" />
						<button class="btn btn-primary" :disabled="updatingName" @click="updateName">Aggiorna</button>
					</div>
				</div>
				<div class="mb-3">
					<label class="form-label">Foto</label>
					<input ref="photoInput" class="form-control" type="file" @change="onPhotoChange" />
					<div class="mt-2">
						<button class="btn btn-secondary" :disabled="updatingPhoto" @click="updatePhoto">Aggiorna foto</button>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>
