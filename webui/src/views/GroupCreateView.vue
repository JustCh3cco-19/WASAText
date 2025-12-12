<script>
import api from "../services/api.js";
import {base64ImageToDataUrl, cropImageSquare} from "../services/files.js";
import sessionStore from "../services/sessionStore.js";

export default {
	data() {
		return {
			errormsg: null,
			creating: false,
			updatingPhoto: false,
			newGroup: {
				name: "",
				members: "",
				imageData: "",
				imageName: "",
				imagePreview: "",
			},
			cropModalOpen: false,
			cropSourceUrl: "",
			cropPreview: "",
			newPhotoCropped: "",
			photoFileName: "",
			cropX: 0,
			cropY: 0,
			cropScale: 1,
		};
	},
	computed: {
		newGroupPhotoStyle() {
			const photo = base64ImageToDataUrl(this.newGroup.imagePreview || this.newGroup.imageData);
			if (photo) {
				return {
					backgroundImage: `url(${photo})`,
					backgroundSize: "cover",
					backgroundPosition: "center",
				};
			}
			return {};
		},
		newGroupAvatarInitial() {
			const name = this.newGroup.name || "G";
			return name.charAt(0).toUpperCase() || "G";
		},
	},
	methods: {
		async onImageChange(event) {
			const inputEl = event?.target;
			const file = inputEl?.files?.[0];
			this.photoFileName = file?.name || "";
			this.newGroup.imageName = file?.name || "";
			if (!file) {
				this.resetPhotoCrop();
				return;
			}
			if (inputEl) inputEl.value = null;
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
				this.cropPreview = `data:image/png;base64,${base64}`;
				this.newGroup.imagePreview = this.cropPreview;
			} catch (e) {
				this.errormsg = e?.toString() || "Errore nel ritaglio";
			}
		},
		async updatePhoto() {
			if (!this.newPhotoCropped) return;
			this.updatingPhoto = true;
			this.errormsg = null;
			this.newGroup.imageData = this.newPhotoCropped;
			this.newGroup.imagePreview = this.cropPreview;
			this.cropModalOpen = false;
			this.updatingPhoto = false;
		},
		resetPhotoCrop() {
			this.newPhotoCropped = "";
			this.cropPreview = "";
			this.cropSourceUrl = "";
			this.cropX = 0;
			this.cropY = 0;
			this.cropScale = 1;
			this.newGroup.imageName = "";
			this.newGroup.imagePreview = "";
			this.newGroup.imageData = "";
			this.photoFileName = "";
			const input = this.$refs.groupImageInput;
			if (input) input.value = null;
			this.cropModalOpen = false;
		},
		async resolveMemberIds(rawMembers) {
			const entries = rawMembers
				.split(",")
				.map((m) => m.trim())
				.filter(Boolean);
			if (entries.length === 0) {
				return [];
			}
			const ids = [];
			for (const entry of entries) {
				try {
					const res = await api.searchUsers(entry);
					const users = res.users || [];
					const match = users.find(
						(u) =>
							(u.name && u.name.toLowerCase() === entry.toLowerCase()) ||
							u.id === entry
					);
					if (!match) {
						throw new Error(`Utente "${entry}" non trovato`);
					}
					ids.push(match.id);
				} catch (err) {
					throw new Error(err?.response?.data?.error || `Utente "${entry}" non trovato`);
				}
			}
			// Deduplicate keeping order
			return ids.filter((id, idx) => ids.indexOf(id) === idx);
		},
		async createGroup() {
			if (!this.newGroup.name.trim()) {
				this.errormsg = "Inserisci un nome per il gruppo";
				return;
			}
			this.creating = true;
			this.errormsg = null;
			try {
				const memberIds = await this.resolveMemberIds(this.newGroup.members);
				const currentUserId = sessionStore.state.user?.id;
				if (currentUserId && !memberIds.includes(currentUserId)) {
					memberIds.push(currentUserId);
				}
				const payload = {
					name: this.newGroup.name.trim(),
					membersJson: JSON.stringify(memberIds),
				};
				if (this.newGroup.imageData) {
					payload.image = this.newGroup.imageData;
				}
				const created = await api.createGroup(payload);
				this.resetPhotoCrop();
				this.newGroup = {name: "", members: "", imageData: "", imageName: "", imagePreview: ""};
				if (created?.id) {
					this.$router.push(`/groups/${created.id}`);
				} else {
					this.$router.push("/groups");
				}
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.message || e.toString();
			}
			this.creating = false;
		},
	},
};
</script>

<template>
  <div class="py-4">
    <div class="d-flex justify-content-between flex-wrap flex-md-nowrap align-items-center pb-2 mb-3 border-bottom">
      <div>
        <h1 class="h3 mb-0">Crea gruppo</h1>
        <p class="text-muted small mb-0">Nome, partecipanti e (opzionale) una foto.</p>
      </div>
      <RouterLink to="/groups" class="btn btn-outline-secondary btn-sm">Torna ai gruppi</RouterLink>
    </div>

    <ErrorMsg v-if="errormsg" :msg="errormsg" />

    <div class="card mb-3 create-card shadow-sm">
      <div class="card-body">
        <div class="row g-3">
          <div class="col-md-6">
            <label class="form-label">Nome gruppo</label>
            <input v-model="newGroup.name" class="form-control" placeholder="Es. Team Backend">
          </div>
          <div class="col-md-6">
            <label class="form-label">Partecipanti</label>
            <input
              v-model="newGroup.members"
              class="form-control"
              placeholder="Nomi separati da virgola. Es: utente1, utente2"
            >
          </div>
          <div class="col-md-8">
            <label class="form-label">Foto gruppo <span class="text-muted small">(opzionale)</span></label>
            <div class="d-flex gap-2 align-items-center flex-wrap">
              <button class="btn btn-outline-secondary" type="button" @click="$refs.groupImageInput?.click()">
                {{ photoFileName || "Scegli foto" }}
              </button>
              <span class="text-muted small">
                {{ photoFileName ? `Selezionato: ${photoFileName}` : "Nessun file selezionato" }}
              </span>
              <span class="text-muted small">Scegli e ritaglia (popup)</span>
              <input ref="groupImageInput" class="d-none" type="file" accept="image/*" @change="onImageChange">
            </div>
          </div>
          <div class="col-md-4">
            <label class="form-label">Anteprima</label>
            <div class="group-photo-preview large" :style="newGroupPhotoStyle">
              <span v-if="!newGroupPhotoStyle.backgroundImage">{{ newGroupAvatarInitial }}</span>
            </div>
          </div>
        </div>
        <div class="mt-3 d-flex justify-content-between align-items-center flex-wrap gap-2">
          <div class="text-muted small">
            I partecipanti inseriti verranno invitati automaticamente; puoi aggiungerne altri in seguito.
          </div>
          <button class="btn btn-primary px-4" :disabled="creating" @click="createGroup">Crea gruppo</button>
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
                <span v-if="updatingPhoto" class="spinner-border spinner-border-sm me-1" role="status" aria-hidden="true" />
                Salva foto
              </button>
            </div>
          </div>
          <div class="row g-3">
            <div class="col-md-6 text-center">
              <div class="crop-preview rounded-circle mx-auto" style="width: 180px; height: 180px; overflow: hidden;">
                <img :src="cropPreview || cropSourceUrl" alt="Preview" style="width: 100%; height: 100%; object-fit: cover;">
              </div>
              <p class="small text-muted mt-2 mb-0">Anteprima ritaglio</p>
            </div>
            <div class="col-md-6">
              <label class="form-label small mb-1">Zoom</label>
              <input v-model.number="cropScale" type="range" class="form-range" min="1" max="2.5" step="0.05" @input="updateCropPreview">
              <label class="form-label small mb-1">Sposta orizzontale</label>
              <input v-model.number="cropX" type="range" class="form-range" min="-0.5" max="0.5" step="0.02" @input="updateCropPreview">
              <label class="form-label small mb-1">Sposta verticale</label>
              <input v-model.number="cropY" type="range" class="form-range" min="-0.5" max="0.5" step="0.02" @input="updateCropPreview">
            </div>
          </div>
        </div>
      </div>
    </div>
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

.group-photo-preview.small {
  width: 56px;
  height: 56px;
  font-size: 22px;
  margin: 0;
}

.create-card {
  border: 1px solid #e9ecef;
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
