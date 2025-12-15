import {createRouter, createWebHashHistory} from 'vue-router';
import sessionStore from '../services/sessionStore.js';
import {setPageTitle} from '../services/pageTitle.js';
import LoginView from '../views/LoginView.vue';
import ConversationsView from '../views/ConversationsView.vue';
import ConversationView from '../views/ConversationView.vue';
import GroupsView from '../views/GroupsView.vue';
import GroupView from '../views/GroupView.vue';
import GroupInfoView from '../views/GroupInfoView.vue';
import GroupCreateView from '../views/GroupCreateView.vue';
import ProfileView from '../views/ProfileView.vue';
import SearchView from '../views/SearchView.vue';

const router = createRouter({
	history: createWebHashHistory(import.meta.env.BASE_URL),
	routes: [
		{path: '/', redirect: '/conversations'},
		{path: '/login', name: 'login', component: LoginView, meta: {public: true, title: "Login"}},
		{path: '/conversations', name: 'conversations', component: ConversationsView, meta: {title: "Chat Private"}},
		{path: '/conversations/:id', name: 'conversation', component: ConversationView, props: true, meta: {title: "Chat"}},
		{path: '/groups', name: 'groups', component: GroupsView, meta: {title: "Gruppi"}},
		{path: '/groups/new', name: 'group-create', component: GroupCreateView, meta: {title: "Crea Gruppo"}},
		{path: '/groups/:id', name: 'group', component: GroupView, props: true, meta: {title: "Gruppo"}},
		{path: '/groups/:id/info', name: 'group-info', component: GroupInfoView, props: true, meta: {title: "Info gruppo"}},
		{path: '/profile', name: 'profile', component: ProfileView, meta: {title: "Profilo"}},
		{path: '/search', name: 'search', component: SearchView, meta: {title: "Cerca Utenti"}},
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

router.afterEach((to) => {
	const matchWithTitle = to.matched.slice().reverse().find((route) => route.meta && route.meta.title);
	const resolvedTitle = matchWithTitle?.meta?.title;
	const pageTitle = typeof resolvedTitle === "function" ? resolvedTitle(to) : resolvedTitle;
	setPageTitle(pageTitle);
});

export default router;
