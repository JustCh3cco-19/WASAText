package api

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/JustCh3cco-19/WASAText/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

var groupNameRegexp = regexp.MustCompile(`^[a-zA-Z0-9_ ]+$`)

func (rt *_router) handleListGroups(w http.ResponseWriter, r *http.Request, _ httprouter.Params, ctx reqcontext.RequestContext) {
	groups, err := rt.db.ListGroups(r.Context(), ctx.AuthenticatedUser.ID)
	if err != nil {
		rt.respondDBError(w, ctx, err, "")
		return
	}
	response := struct {
		Groups []groupResponse `json:"groups"`
	}{
		Groups: make([]groupResponse, 0, len(groups)),
	}
	for _, g := range groups {
		response.Groups = append(response.Groups, toGroupResponse(g))
	}
	writeJSON(w, http.StatusOK, response)
}

func (rt *_router) handleCreateGroup(w http.ResponseWriter, r *http.Request, _ httprouter.Params, ctx reqcontext.RequestContext) {
	if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid multipart payload")
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if len(name) < 3 || len(name) > 50 || !groupNameRegexp.MatchString(name) {
		writeError(w, http.StatusBadRequest, "Invalid group name")
		return
	}
	membersRaw := strings.TrimSpace(r.FormValue("membersJson"))
	if membersRaw == "" {
		writeError(w, http.StatusBadRequest, "membersJson is required")
		return
	}
	var members []string
	if err := json.Unmarshal([]byte(membersRaw), &members); err != nil {
		writeError(w, http.StatusBadRequest, "membersJson must be a JSON array of user IDs")
		return
	}
	members = sanitizeMemberIDs(members)
	if len(members) == 0 {
		writeError(w, http.StatusBadRequest, "membersJson cannot be empty")
		return
	}
	if !containsString(members, ctx.AuthenticatedUser.ID) {
		members = append(members, ctx.AuthenticatedUser.ID)
	}
	image := strings.TrimSpace(r.FormValue("image"))
	if image == "" {
		writeError(w, http.StatusBadRequest, "image payload is required")
		return
	}

	group, err := rt.db.CreateGroupConversation(r.Context(), name, image, members)
	if err != nil {
		rt.respondDBError(w, ctx, err, "")
		return
	}
	writeJSON(w, http.StatusCreated, toGroupResponse(group))
}

func (rt *_router) handleGetGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	groupID := strings.TrimSpace(ps.ByName("groupId"))
	if groupID == "" {
		writeError(w, http.StatusBadRequest, "Group ID is required")
		return
	}
	group, err := rt.db.GetGroup(r.Context(), ctx.AuthenticatedUser.ID, groupID)
	if err != nil {
		rt.respondDBError(w, ctx, err, "Group not found")
		return
	}
	writeJSON(w, http.StatusOK, toGroupResponse(group))
}

func (rt *_router) handleLeaveGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	groupID := strings.TrimSpace(ps.ByName("groupId"))
	if groupID == "" {
		writeError(w, http.StatusBadRequest, "Group ID is required")
		return
	}
	if err := rt.db.LeaveGroup(r.Context(), ctx.AuthenticatedUser.ID, groupID); err != nil {
		rt.respondDBError(w, ctx, err, "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) handleAddGroupMember(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	groupID := strings.TrimSpace(ps.ByName("groupId"))
	if groupID == "" {
		writeError(w, http.StatusBadRequest, "Group ID is required")
		return
	}
	var payload struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}
	payload.UserID = strings.TrimSpace(payload.UserID)
	if payload.UserID == "" {
		writeError(w, http.StatusBadRequest, "userId is required")
		return
	}
	if err := rt.db.AddGroupMember(r.Context(), ctx.AuthenticatedUser.ID, groupID, payload.UserID); err != nil {
		rt.respondDBError(w, ctx, err, "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) handleUpdateGroupName(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	groupID := strings.TrimSpace(ps.ByName("groupId"))
	if groupID == "" {
		writeError(w, http.StatusBadRequest, "Group ID is required")
		return
	}
	var payload struct {
		GroupName string `json:"groupName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}
	payload.GroupName = strings.TrimSpace(payload.GroupName)
	if len(payload.GroupName) < 3 || len(payload.GroupName) > 50 || !groupNameRegexp.MatchString(payload.GroupName) {
		writeError(w, http.StatusBadRequest, "Invalid group name")
		return
	}
	group, err := rt.db.UpdateGroupName(r.Context(), ctx.AuthenticatedUser.ID, groupID, payload.GroupName)
	if err != nil {
		rt.respondDBError(w, ctx, err, "")
		return
	}
	writeJSON(w, http.StatusOK, toGroupResponse(group))
}

func (rt *_router) handleUpdateGroupPhoto(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	groupID := strings.TrimSpace(ps.ByName("groupId"))
	if groupID == "" {
		writeError(w, http.StatusBadRequest, "Group ID is required")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBinaryPayload))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Unable to read payload")
		return
	}
	photo := strings.TrimSpace(string(body))
	if photo == "" {
		writeError(w, http.StatusBadRequest, "Photo payload cannot be empty")
		return
	}
	if err := rt.db.UpdateGroupPhoto(r.Context(), ctx.AuthenticatedUser.ID, groupID, photo); err != nil {
		rt.respondDBError(w, ctx, err, "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Group photo updated successfully",
	})
}

func sanitizeMemberIDs(ids []string) []string {
	seen := make(map[string]struct{})
	res := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		res = append(res, id)
	}
	return res
}

func containsString(list []string, target string) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}
