package api

import "net/http"

// Handler returns an instance of httprouter.Router that handle APIs registered here
func (rt *_router) Handler() http.Handler {
	// Public routes
	rt.router.POST("/session", rt.wrap(rt.handleLogin))

	// Authenticated routes
	rt.router.PUT("/users/photo", rt.wrapAuthenticated(rt.handleUpdateUserPhoto))
	rt.router.PUT("/users/name", rt.wrapAuthenticated(rt.handleUpdateUserName))

	rt.router.GET("/conversations", rt.wrapAuthenticated(rt.handleListConversations))
	rt.router.POST("/conversations", rt.wrapAuthenticated(rt.handleStartConversation))
	rt.router.GET("/conversations/:conversationId", rt.wrapAuthenticated(rt.handleGetConversation))
	rt.router.POST("/conversations/:conversationId/message", rt.wrapAuthenticated(rt.handleSendMessage))
	rt.router.POST("/conversations/:conversationId/message/:messageId/forward", rt.wrapAuthenticated(rt.handleForwardMessage))
	rt.router.DELETE("/conversations/:conversationId/message/:messageId", rt.wrapAuthenticated(rt.handleDeleteMessage))
	rt.router.POST("/conversations/:conversationId/message/:messageId/comment", rt.wrapAuthenticated(rt.handleCommentMessage))
	rt.router.DELETE("/conversations/:conversationId/message/:messageId/comment", rt.wrapAuthenticated(rt.handleUncommentMessage))

	rt.router.GET("/search", rt.wrapAuthenticated(rt.handleSearchUsers))

	rt.router.GET("/groups", rt.wrapAuthenticated(rt.handleListGroups))
	rt.router.POST("/groups", rt.wrapAuthenticated(rt.handleCreateGroup))
	rt.router.GET("/groups/:groupId", rt.wrapAuthenticated(rt.handleGetGroup))
	rt.router.DELETE("/groups/:groupId", rt.wrapAuthenticated(rt.handleLeaveGroup))
	rt.router.POST("/groups/:groupId", rt.wrapAuthenticated(rt.handleAddGroupMember))
	rt.router.PUT("/groups/:groupId/name", rt.wrapAuthenticated(rt.handleUpdateGroupName))
	rt.router.PUT("/groups/:groupId/photo", rt.wrapAuthenticated(rt.handleUpdateGroupPhoto))

	// Special routes
	rt.router.GET("/liveness", rt.liveness)

	return rt.router
}
