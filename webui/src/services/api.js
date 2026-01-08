import axios from "./axios.js";

const defaultHeaders = (contentType) => contentType ? {"Content-Type": contentType} : {};

export default {
	async login(name) {
		const {data} = await axios.post("/session", {name});
		return data;
	},

	async getConversations() {
		const {data} = await axios.get("/conversations");
		return data;
	},

	async startConversation(recipientId) {
		const {data} = await axios.post("/conversations", {recipientId});
		return data;
	},

	async getConversation(conversationId) {
		const {data} = await axios.get(`/conversations/${conversationId}`);
		return data;
	},

	async sendMessage(conversationId, {content, attachment, replyTo}) {
		const formData = new FormData();
		if (content) {
			formData.append("content", content);
		}
		if (attachment) {
			formData.append("attachment", attachment);
		}
		if (replyTo) {
			formData.append("replyTo", replyTo);
		}
		const {data} = await axios.post(
			`/conversations/${conversationId}/message`,
			formData,
			{headers: {"Content-Type": "multipart/form-data"}}
		);
		return data;
	},

	async forwardMessage(conversationId, messageId, targetConversationId, targetUserId) {
		const payload = {};
		if (targetConversationId) {
			payload.targetConversationId = targetConversationId;
		}
		if (targetUserId) {
			payload.targetUserId = targetUserId;
		}
		const {data} = await axios.post(
			`/conversations/${conversationId}/message/${messageId}/forward`,
			payload
		);
		return data;
	},

	async deleteMessage(conversationId, messageId) {
		await axios.delete(`/conversations/${conversationId}/message/${messageId}`);
	},

	async commentMessage(conversationId, messageId, content) {
		await axios.post(
			`/conversations/${conversationId}/message/${messageId}/comment`,
			{content}
		);
	},

	async deleteComment(conversationId, messageId) {
		await axios.delete(`/conversations/${conversationId}/message/${messageId}/comment`);
	},

	async searchUsers(username) {
		const {data} = await axios.get("/search", {params: {username}});
		return data;
	},

	async getGroups() {
		const {data} = await axios.get("/groups");
		return data;
	},

	async createGroup({name, membersJson, image}) {
		const formData = new FormData();
		if (name) {
			formData.append("name", name);
		}
		if (membersJson) {
			formData.append("membersJson", membersJson);
		}
		if (image) {
			formData.append("image", image);
		}
		const {data} = await axios.post("/groups", formData, {
			headers: {"Content-Type": "multipart/form-data"}
		});
		return data;
	},

	async getGroup(groupId) {
		const {data} = await axios.get(`/groups/${groupId}`);
		return data;
	},

	async leaveGroup(groupId) {
		await axios.delete(`/groups/${groupId}`);
	},

	async addToGroup(groupId, userId) {
		await axios.post(`/groups/${groupId}`, {userId});
	},

	async updateGroupName(groupId, name) {
		const {data} = await axios.put(
			`/groups/${groupId}/name`,
			{name}
		);
		return data;
	},

	async updateGroupPhoto(groupId, image, contentType = "image/png") {
		const {data} = await axios.put(
			`/groups/${groupId}/photo`,
			image,
			{headers: defaultHeaders(contentType)}
		);
		return data;
	},

	async setMyName(name) {
		const {data} = await axios.put("/users/name", {name});
		return data;
	},

	async setMyPhoto(photo, contentType = "image/png") {
		const {data} = await axios.put(
			"/users/photo",
			photo,
			{headers: defaultHeaders(contentType)}
		);
		return data;
	},
};
