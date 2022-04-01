package handlers

import (
	"encoding/json"
	"github.com/matryer/way"
	"net/http"
	"social/internal/services"
	"strconv"
)

type createUserInput struct {
	Email, Username string
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var input createUserInput
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.CreateUser(r.Context(), input.Email, input.Username)

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	w.WriteHeader(http.StatusNoContent)

}

func (h *Handler) toggleFollow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	username := way.Param(ctx, "username")

	out, err := h.ToggleFollow(ctx, username)
	if err != nil {
		respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	respond(w, out, http.StatusOK)
}

func (h *Handler) getUserProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	username := way.Param(ctx, "username")

	userProfile, err := h.GetUserProfile(ctx, username)

	if err != nil {
		respondError(w, err)
		return
	}
	respond(w, userProfile, http.StatusOK)

}

func (h *Handler) getUserProfiles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	search := q.Get("search")
	first, _ := strconv.Atoi(q.Get("first"))
	after := q.Get("after")

	result, err := h.GetUsers(ctx, search, first, after)

	if err != nil {
		respondError(w, err)
		return
	}

	respond(w, result, http.StatusOK)

}

func (h *Handler) getFollowers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	username := q.Get("username")
	first, _ := strconv.Atoi(q.Get("first"))
	after := q.Get("after")

	result, err := h.GetFollowers(ctx, username, first, after)

	if err != nil {
		respondError(w, err)
		return
	}

	respond(w, result, http.StatusOK)

}

func (h *Handler) getFollows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	username := q.Get("username")
	first, _ := strconv.Atoi(q.Get("first"))
	after := q.Get("after")

	result, err := h.GetFollowees(ctx, username, first, after)

	if err != nil {
		respondError(w, err)
		return
	}

	respond(w, result, http.StatusOK)

}

func (h *Handler) updateAvatar(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, services.MaxAvatarBytes)
	defer r.Body.Close()

	avatarUrl, err := h.UpdateAvatar(r.Context(), r.Body)
	if err != nil {
		respondError(w, err)
		return
	}
	respond(w, avatarUrl, http.StatusOK)
}
