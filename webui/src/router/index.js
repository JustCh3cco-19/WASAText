import {createRouter, createWebHashHistory} from 'vue-router';
import sessionStore from '../services/sessionStore.js';
import LoginView from '../views/LoginView.vue';
import ConversationsView from '../views/ConversationsView.vue';
import ConversationView from '../views/ConversationView.vue';
import GroupsView from '../views/GroupsView.vue';
import GroupView from '../views/GroupView.vue';
import ProfileView from '../views/ProfileView.vue';
import SearchView from '../views/SearchView.vue';

const router = createRouter({
	history: createWebHashHistory(import.meta.env.BASE_URL),
	routes: [
		{path: '/', redirect: '/conversations'},
		{path: '/login', name: 'login', component: LoginView, meta: {public: true}},
		{path: '/conversations', name: 'conversations', component: ConversationsView},
		{path: '/conversations/:id', name: 'conversation', component: ConversationView, props: true},
		{path: '/groups', name: 'groups', component: GroupsView},
		{path: '/groups/:id', name: 'group', component: GroupView, props: true},
		{path: '/profile', name: 'profile', component: ProfileView},
		{path: '/search', name: 'search', component: SearchView},
		{path: '/:pathMatch(.*)*', redirect: '/conversations'},
	]
});

router.beforeEach((to, from, next) => {
	if (!to.meta.public && !sessionStore.isAuthenticated()) {
		return next({path: '/login', query: {redirect: to.fullPath}});
	}
	if (to.name === 'login' && sessionStore.isAuthenticated()) {
		return next('/conversations');
	}
	return next();
});

export default router;
