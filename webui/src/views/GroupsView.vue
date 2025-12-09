<script>
import api from "../services/api.js";
import {fileToBase64} from "../services/files.js";
import sessionStore from "../services/sessionStore.js";

export default {
	data() {
		return {
			groups: [],
			errormsg: null,
			loading: false,
			creating: false,
			newGroup: {
				name: "",
				members: "",
				imageData: "",
			},
		};
	},
	methods: {
		async refresh() {
			this.loading = true;
			this.errormsg = null;
			try {
				const res = await api.getGroups();
				this.groups = res.groups || [];
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.toString();
			}
			this.loading = false;
		},
		async onImageChange(event) {
			const file = event?.target?.files?.[0];
			this.newGroup.imageData = file ? await fileToBase64(file) : "";
		},
		async createGroup() {
			if (!this.newGroup.name.trim()) return;
			this.creating = true;
			this.errormsg = null;
			try {
				const membersArray = this.newGroup.members
					.split(",")
					.map((m) => m.trim())
					.filter(Boolean);
				if (sessionStore.state.token && !membersArray.includes(sessionStore.state.token)) {
					membersArray.push(sessionStore.state.token);
				}
				const payload = {
					name: this.newGroup.name.trim(),
					membersJson: JSON.stringify(membersArray),
					image: this.newGroup.imageData,
				};
				const created = await api.createGroup(payload);
				this.newGroup = {name: "", members: "", imageData: ""};
				await this.refresh();
				if (created?.id) {
					this.$router.push(`/groups/${created.id}`);
				}
			} catch (e) {
				this.errormsg = e?.response?.data?.error || e.toString();
			}
			this.creating = false;
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
			<h1 class="h3 mb-0">Gruppi</h1>
		</div>

		<ErrorMsg v-if="errormsg" :msg="errormsg" />

		<div class="card mb-3">
			<div class="card-body">
				<h5 class="card-title">Crea gruppo</h5>
				<div class="row g-2">
					<div class="col-md-4">
						<input class="form-control" placeholder="Nome gruppo" v-model="newGroup.name" />
					</div>
					<div class="col-md-4">
						<input class="form-control" placeholder="Membri separati da virgola" v-model="newGroup.members" />
					</div>
					<div class="col-md-4">
						<input class="form-control" type="file" @change="onImageChange" />
					</div>
				</div>
				<div class="mt-3 d-flex justify-content-end">
					<button class="btn btn-primary" :disabled="creating" @click="createGroup">Crea</button>
				</div>
			</div>
		</div>

		<LoadingSpinner :loading="loading">
			<div v-if="groups.length === 0" class="text-muted">Nessun gruppo.</div>
			<div class="row row-cols-1 row-cols-md-2 g-3">
				<div class="col" v-for="group in groups" :key="group.id">
					<div class="card h-100">
						<div class="card-body d-flex flex-column">
							<div class="d-flex align-items-center justify-content-between">
								<h5 class="card-title mb-0">{{ group.name || group.id }}</h5>
								<RouterLink :to="`/groups/${group.id}`" class="btn btn-sm btn-primary">Apri</RouterLink>
							</div>
							<p class="text-muted small mt-2 mb-1">Partecipanti: {{ (group.members || []).join(", ") }}</p>
						</div>
					</div>
				</div>
			</div>
		</LoadingSpinner>
	</div>
</template>
