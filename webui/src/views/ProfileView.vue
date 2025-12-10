<script>
import api from "../services/api.js";
import sessionStore from "../services/sessionStore.js";
import {cropImageSquare, base64ImageToDataUrl} from "../services/files.js";

const STATUS_KEY = "userStatus";

export default {
	data() {
		return {
			errormsg: null,
			loading: false,
			newName: sessionStore.state.user?.name || "",
			newPhotoCropped: "",
			photoPreview: "",
			cropSourceUrl: "",
			cropModalOpen: false,
			cropX: 0,
			cropY: 0,
			cropScale: 1,
			updatingName: false,
			updatingPhoto: false,
			status: localStorage.getItem(STATUS_KEY) || "Disponibile",
		};
	},
	computed: {
		displayName() {
			return sessionStore.state.user?.name || "Tu";
		},
		avatarStyle() {
			const photo = base64ImageToDataUrl(sessionStore.state.user?.photo || "");
			if (photo) {
				return {
					backgroundImage: `url(${photo})`,
					backgroundSize: "cover",
					backgroundPosition: "center",
				};
			}
			return {};
		},
		avatarInitial() {
			const name = this.displayName || "";
			return name.charAt(0).toUpperCase() || "U";
		},
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
			if (!file) {
				this.resetPhotoCrop();
				return;
			}
			const reader = new FileReader();
			reader.onload = () => {
				this.cropSourceUrl = reader.result;
				this.cropModalOpen = true;
				this.updateCropPreview();
			};
			reader.onerror = () => {
				this.errormsg = "Errore nel leggere il file";
			};
			reader.readAsDataURL(file);
		},
		async updatePhoto() {
			if (!this.newPhotoCropped) return;
			this.updatingPhoto = true;
			this.errormsg = null;
			try {
				const user = await api.setMyPhoto(this.newPhotoCropped);
				sessionStore.setUser(user);
				this.resetPhotoCrop();
				const input = this.$refs.photoInput;
				if (input) input.value = null;
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.toString();
			}
			this.updatingPhoto = false;
		},
		async updateCropPreview() {
			if (!this.cropSourceUrl) return;
			try {
				const base64 = await cropImageSquare(this.cropSourceUrl, {
					size: 320,
					offsetX: this.cropX,
					offsetY: this.cropY,
					scale: this.cropScale,
				});
				this.newPhotoCropped = base64;
				this.photoPreview = `data:image/png;base64,${base64}`;
			} catch (e) {
				this.errormsg = e?.toString() || "Errore nel ritaglio";
			}
		},
		resetPhotoCrop() {
			this.newPhotoCropped = "";
			this.photoPreview = "";
			this.cropSourceUrl = "";
			this.cropModalOpen = false;
			this.cropX = 0;
			this.cropY = 0;
			this.cropScale = 1;
		},
		updateStatus() {
			this.status = this.status.trim() || "Disponibile";
			localStorage.setItem(STATUS_KEY, this.status);
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

    <div class="card mb-3 profile-card text-center">
      <div class="card-body">
        <div class="avatar mx-auto mb-3" :style="avatarStyle">
          <span v-if="!avatarStyle.backgroundImage">{{ avatarInitial }}</span>
        </div>
        <h2 class="h4 mb-1">{{ displayName }}</h2>
        <div class="d-flex justify-content-center align-items-center gap-2 flex-wrap mb-3">
          <input v-model="status" class="form-control w-auto" style="min-width: 220px" maxlength="80">
          <button class="btn btn-outline-primary btn-sm" @click="updateStatus">Aggiorna stato</button>
        </div>
        <div class="text-muted small">Stato visibile solo a te per ora.</div>
      </div>
    </div>

    <div class="card">
      <div class="card-body">
        <h5 class="card-title">Impostazioni account</h5>
        <div class="mb-3">
          <label class="form-label">Nome</label>
          <div class="input-group">
            <input v-model="newName" class="form-control">
            <button class="btn btn-primary" :disabled="updatingName" @click="updateName">Aggiorna</button>
          </div>
        </div>
        <div class="mb-3">
          <label class="form-label">Foto</label>
          <div class="d-flex gap-2 align-items-center">
            <input ref="photoInput" class="form-control" type="file" @change="onPhotoChange">
            <span class="text-muted small">Scegli e ritaglia (popup)</span>
          </div>
        </div>
      </div>
    </div>

    <div v-if="cropModalOpen" class="crop-modal">
      <div class="crop-dialog card shadow">
        <div class="card-body">
          <div class="d-flex justify-content-between align-items-center mb-2">
            <h5 class="card-title mb-0">Ritaglia foto</h5>
            <div class="d-flex gap-2">
              <button class="btn btn-sm btn-outline-secondary" @click="resetPhotoCrop">Annulla</button>
              <button
                class="btn btn-sm btn-primary"
                :disabled="updatingPhoto || !newPhotoCropped"
                @click="updatePhoto"
              >
                <span v-if="updatingPhoto" class="spinner-border spinner-border-sm me-1" role="status" aria-hidden="true"></span>
                Salva foto
              </button>
            </div>
          </div>
          <div class="row g-3">
            <div class="col-md-6 text-center">
              <div class="crop-preview rounded-circle mx-auto" style="width: 180px; height: 180px; overflow: hidden;">
                <img :src="photoPreview || cropSourceUrl" alt="Preview" style="width: 100%; height: 100%; object-fit: cover;">
              </div>
              <p class="small text-muted mt-2 mb-0">Anteprima ritaglio</p>
            </div>
            <div class="col-md-6">
              <label class="form-label small mb-1">Zoom</label>
              <input type="range" class="form-range" min="1" max="2.5" step="0.05" v-model.number="cropScale" @input="updateCropPreview">
              <label class="form-label small mb-1">Sposta orizzontale</label>
              <input type="range" class="form-range" min="-0.5" max="0.5" step="0.02" v-model.number="cropX" @input="updateCropPreview">
              <label class="form-label small mb-1">Sposta verticale</label>
              <input type="range" class="form-range" min="-0.5" max="0.5" step="0.02" v-model.number="cropY" @input="updateCropPreview">
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.profile-card .avatar {
  width: 110px;
  height: 110px;
  border-radius: 50%;
  background-color: #f1f3f5;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 36px;
  color: #6c757d;
  overflow: hidden;
}

.crop-modal {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1050;
}

.crop-dialog {
  width: min(720px, 95vw);
}
</style>
