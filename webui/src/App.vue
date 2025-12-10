<script setup>
import {computed} from "vue";
import {RouterLink, RouterView, useRouter} from "vue-router";
import sessionStore from "./services/sessionStore.js";

const router = useRouter();
const isAuthenticated = computed(() => sessionStore.isAuthenticated());
const userLabel = computed(() => sessionStore.state.user?.name || sessionStore.state.token || "Ospite");

function doLogout() {
	sessionStore.logout();
	router.push("/login");
}
</script>

<template>
  <header class="navbar navbar-dark sticky-top bg-dark flex-md-nowrap p-0 shadow">
    <RouterLink class="navbar-brand col-md-3 col-lg-2 me-0 px-3 fs-6" to="/conversations">
      WASAText
    </RouterLink>
    <button class="navbar-toggler position-absolute d-md-none collapsed" type="button" data-bs-toggle="collapse" data-bs-target="#sidebarMenu" aria-controls="sidebarMenu" aria-expanded="false" aria-label="Toggle navigation">
      <span class="navbar-toggler-icon" />
    </button>
    <div class="w-100 text-end pe-3 text-white small">
      <span v-if="isAuthenticated">Ciao, {{ userLabel }}</span>
      <button v-if="isAuthenticated" class="btn btn-sm btn-outline-light ms-2" @click="doLogout">Logout</button>
    </div>
  </header>

  <div class="container-fluid">
    <div class="row">
      <nav id="sidebarMenu" class="col-md-3 col-lg-2 d-md-block bg-light sidebar collapse">
        <div class="position-sticky pt-3 sidebar-sticky">
          <h6 class="sidebar-heading d-flex justify-content-between align-items-center px-3 mt-4 mb-1 text-muted text-uppercase">
            <span>Navigazione</span>
          </h6>
          <ul class="nav flex-column">
            <li class="nav-item">
              <RouterLink to="/conversations" class="nav-link">
                <svg class="feather"><use href="/feather-sprite-v4.29.0.svg#message-square" /></svg>
                Chat
              </RouterLink>
            </li>
            <li class="nav-item">
              <RouterLink to="/groups" class="nav-link">
                <svg class="feather"><use href="/feather-sprite-v4.29.0.svg#users" /></svg>
                Gruppi
              </RouterLink>
            </li>
            <li class="nav-item">
              <RouterLink to="/search" class="nav-link">
                <svg class="feather"><use href="/feather-sprite-v4.29.0.svg#search" /></svg>
                Cerca utenti
              </RouterLink>
            </li>
            <li class="nav-item">
              <RouterLink to="/profile" class="nav-link">
                <svg class="feather"><use href="/feather-sprite-v4.29.0.svg#user" /></svg>
                Profilo
              </RouterLink>
            </li>
            <li v-if="!isAuthenticated" class="nav-item">
              <RouterLink to="/login" class="nav-link">
                <svg class="feather"><use href="/feather-sprite-v4.29.0.svg#log-in" /></svg>
                Login
              </RouterLink>
            </li>
          </ul>
        </div>
      </nav>

      <main class="col-md-9 ms-sm-auto col-lg-10 px-md-4">
        <RouterView />
      </main>
    </div>
  </div>
</template>
