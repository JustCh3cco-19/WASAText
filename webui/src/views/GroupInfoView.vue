<script>
import api from "../services/api.js";
import {fileToBase64, base64ImageToDataUrl, cropImageSquare} from "../services/files.js";

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
			errormsg: null,
			loading: false,
			updatingName: false,
			newName: "",
			updatingPhoto: false,
			photoFileName: "",
			photoCrop: {
				sourceUrl: "",
				cropped: "",
				preview: "",
				scale: 1,
				x: 0,
				y: 0,
			},
			cropModalOpen: false,
			newUserQuery: "",
			addingUser: false,
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
		memberList() {
			if (this.group?.memberDetails?.length) {
				return this.group.memberDetails;
			}
			const count = (this.group?.members || []).length;
			return Array.from({length: count}, (_, idx) => ({name: `Membro ${idx + 1}`}));
		},
	},
	mounted() {
		this.refresh();
	},
	methods: {
		async refresh() {
			this.loading = true;
			this.errormsg = null;
			try {
				const res = await api.getGroup(this.id);
				this.group = res;
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
			const inputEl = this.$refs.groupPhotoInput || event?.target;
			const file = inputEl?.files?.[0];
			this.photoFileName = file?.name || "";
			if (!file) {
				this.resetPhotoCrop();
				if (inputEl) inputEl.value = null;
				return;
			}
			if (inputEl) inputEl.value = null;
			this.errormsg = null;
			const reader = new FileReader();
			reader.onload = () => {
				this.photoCrop.sourceUrl = reader.result;
				this.photoCrop.preview = reader.result;
				this.cropModalOpen = true;
				this.updateCropPreview();
			};
			reader.onerror = () => {
				this.errormsg = "Errore nel leggere il file";
			};
			reader.readAsDataURL(file);
		},
		async updatePhoto() {
			if (!this.photoCrop.cropped) return;
			this.updatingPhoto = true;
			this.errormsg = null;
			try {
				await api.updateGroupPhoto(this.id, this.photoCrop.cropped, "image/png");
				await this.refresh();
				this.resetPhotoCrop();
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.toString();
			}
			this.updatingPhoto = false;
		},
		async updateCropPreview() {
			if (!this.photoCrop.sourceUrl) return;
			try {
				const base64 = await cropImageSquare(this.photoCrop.sourceUrl, {
					size: 320,
					offsetX: this.photoCrop.x,
					offsetY: this.photoCrop.y,
					scale: this.photoCrop.scale,
				});
				this.photoCrop.cropped = base64;
				this.photoCrop.preview = `data:image/png;base64,${base64}`;
			} catch (e) {
				this.errormsg = e?.toString() || "Errore nel ritaglio";
			}
		},
		resetPhotoCrop() {
			this.photoCrop = {
				sourceUrl: "",
				cropped: "",
				preview: "",
				scale: 1,
				x: 0,
				y: 0,
			};
			this.cropModalOpen = false;
			this.photoFileName = "";
			const input = this.$refs.groupPhotoInput;
			if (input) input.value = null;
		},
		async resolveUserId(query) {
			const trimmed = (query || "").trim();
			if (!trimmed) {
				throw new Error("Inserisci un nome utente");
			}
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
	},
};
</script>

<template>
  <div class="py-4">
    <div class="d-flex justify-content-between flex-wrap flex-md-nowrap align-items-center pb-2 mb-3 border-bottom">
      <div>
        <h1 class="h3 mb-0">Info gruppo</h1>
        <p class="text-muted small mb-0">Gestisci nome, foto e membri di {{ group?.name || "questo gruppo" }}.</p>
      </div>
      <div class="d-flex gap-2">
        <RouterLink :to="`/groups/${id}`" class="btn btn-outline-secondary btn-sm">Torna alla chat</RouterLink>
      </div>
    </div>

    <ErrorMsg v-if="errormsg" :msg="errormsg" />

    <LoadingSpinner :loading="loading">
      <div class="row g-3">
        <div class="col-lg-6">
          <div class="card h-100">
            <div class="card-body">
              <div class="d-flex align-items-center gap-3 mb-3 flex-wrap">
                <div class="group-photo-preview large" :style="groupAvatarStyle">
                  <span v-if="!groupAvatarStyle.backgroundImage">{{ groupAvatarInitial }}</span>
                </div>
                <div>
                  <h5 class="card-title mb-1">{{ group?.name || "Gruppo" }}</h5>
                  <p class="text-muted small mb-0">Foto attuale del gruppo</p>
                </div>
              </div>

              <div class="mb-3">
                <label class="form-label">Nome</label>
                <div class="input-group">
                  <input v-model="newName" class="form-control">
                  <button class="btn btn-primary" :disabled="updatingName" @click="updateName">Salva</button>
                </div>
              </div>

              <div class="mb-3">
                <label class="form-label">Foto</label>
                <div class="d-flex gap-2 align-items-center flex-wrap">
                  <button class="btn btn-outline-secondary" type="button" :disabled="updatingPhoto" @click="$refs.groupPhotoInput?.click()">
                    {{ photoFileName || "Scegli foto" }}
                  </button>
                  <span class="text-muted small">
                    {{ photoFileName ? `Selezionato: ${photoFileName}` : "Nessun file selezionato" }}
                  </span>
                  <input ref="groupPhotoInput" class="d-none" type="file" accept="image/*" @change="onPhotoChange">
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="col-lg-6">
          <div class="card h-100">
            <div class="card-body">
              <div class="d-flex justify-content-between align-items-center mb-2">
                <h5 class="card-title mb-0">Membri ({{ memberList.length }})</h5>
              </div>
              <div class="mb-3">
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
              <div class="list-group">
                <div v-for="member in memberList" :key="member.id || member" class="list-group-item">
                  <strong>{{ member.name || member.id }}</strong>
                </div>
                <div v-if="memberList.length === 0" class="text-muted">Nessun membro.</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-if="cropModalOpen" class="crop-modal">
        <div class="crop-dialog card shadow">
          <div class="card-body">
            <div class="d-flex justify-content-between align-items-center mb-2">
              <h5 class="card-title mb-0">Ritaglia foto gruppo</h5>
              <div class="d-flex gap-2">
                <button class="btn btn-sm btn-outline-secondary" @click="resetPhotoCrop">Annulla</button>
                <button
                  class="btn btn-sm btn-primary"
                  :disabled="updatingPhoto || !photoCrop.cropped"
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
                  <img :src="photoCrop.preview || photoCrop.sourceUrl" alt="Preview" style="width: 100%; height: 100%; object-fit: cover;">
                </div>
                <p class="small text-muted mt-2 mb-0">Anteprima ritaglio</p>
              </div>
              <div class="col-md-6">
                <label class="form-label small mb-1">Zoom</label>
                <input type="range" class="form-range" min="1" max="2.5" step="0.05" v-model.number="photoCrop.scale" @input="updateCropPreview">
                <label class="form-label small mb-1">Sposta orizzontale</label>
                <input type="range" class="form-range" min="-0.5" max="0.5" step="0.02" v-model.number="photoCrop.x" @input="updateCropPreview">
                <label class="form-label small mb-1">Sposta verticale</label>
                <input type="range" class="form-range" min="-0.5" max="0.5" step="0.02" v-model.number="photoCrop.y" @input="updateCropPreview">
              </div>
            </div>
          </div>
        </div>
      </div>
    </LoadingSpinner>
  </div>
</template>

<style scoped>
.group-photo-preview {
  width: 110px;
  height: 110px;
  border-radius: 50%;
  background-color: #f1f3f5;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 32px;
  color: #6c757d;
  overflow: hidden;
}

.group-photo-preview.large {
  width: 120px;
  height: 120px;
  font-size: 36px;
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
