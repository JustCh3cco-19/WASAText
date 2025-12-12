<script>
import api from "../services/api.js";

export default {
	data() {
		return {
			groups: [],
			errormsg: null,
			loading: false,
		};
	},
	mounted() {
		this.refresh();
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
		formatMembers(group) {
			if (group?.memberDetails?.length) {
				const names = group.memberDetails.map((u) => u?.name).filter(Boolean);
				if (names.length) return names.join(", ");
				return `${group.memberDetails.length} membri`;
			}
			const count = (group?.members || []).length;
			if (count) return `${count} membri`;
			return "Partecipanti non disponibili";
		},
	},
};
</script>

<template>
  <div class="py-4">
    <div class="d-flex justify-content-between flex-wrap flex-md-nowrap align-items-center pb-2 mb-3 border-bottom">
      <h1 class="h3 mb-0">Gruppi</h1>
      <RouterLink to="/groups/new" class="btn btn-primary btn-sm">Crea gruppo</RouterLink>
    </div>

    <ErrorMsg v-if="errormsg" :msg="errormsg" />

    <LoadingSpinner :loading="loading">
      <div v-if="groups.length === 0" class="text-muted">Nessun gruppo.</div>
      <div v-else class="row row-cols-1 row-cols-md-2 g-3">
        <div v-for="group in groups" :key="group.id" class="col">
          <div class="card h-100">
            <div class="card-body d-flex flex-column">
              <div class="d-flex align-items-center justify-content-between">
                <h5 class="card-title mb-0">{{ group.name || "Gruppo" }}</h5>
                <RouterLink :to="`/groups/${group.id}`" class="btn btn-sm btn-primary">Apri</RouterLink>
              </div>
              <p class="text-muted small mt-2 mb-1">Partecipanti: {{ formatMembers(group) }}</p>
            </div>
          </div>
        </div>
      </div>
    </LoadingSpinner>
  </div>
</template>
