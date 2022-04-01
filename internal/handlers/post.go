package handlers

import (
	"encoding/json"
	"github.com/matryer/way"
	"net/http"
	"strconv"
)

type createPostInput struct {
	Content   string
	SpoilerOf *string
	NSFW      bool
}

func (h *Handler) createPost(w http.ResponseWriter, r *http.Request) {
	var input createPostInput
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, err)
		return
	}
	result, err := h.CreatePost(r.Context(), input.Content, input.SpoilerOf, input.NSFW)
	if err != nil {
		respondError(w, err)
		return
	}

	respond(w, result, http.StatusCreated)
}

func (h *Handler) togglePostLike(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	postId, err := strconv.ParseInt(way.Param(ctx, "postId"), 10, 64)

	result, err := h.TogglePostLike(ctx, postId)
	if err != nil {
		respondError(w, err)
		return
	}

	respond(w, result, http.StatusOK)

}

func (h *Handler) getPostsForUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userId, err := strconv.ParseInt(way.Param(ctx, "id"), 10, 64)

	result, err := h.GetPostsByUserId(ctx, userId)
	if err != nil {
		respondError(w, err)
		return
	}

	respond(w, result, http.StatusOK)

}

func (h *Handler) getPostById(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	postId, err := strconv.ParseInt(way.Param(ctx, "postId"), 10, 64)

	result, err := h.GetPostById(ctx, postId)
	if err != nil {
		respondError(w, err)
		return
	}

	respond(w, result, http.StatusOK)

}

func (h *Handler) getPosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	result, err := h.GetPosts(ctx)
	if err != nil {
		respondError(w, err)
		return
	}

	respond(w, result, http.StatusOK)

}

func (h *Handler) getMyPosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	result, err := h.GetMyPosts(ctx)
	if err != nil {
		respondError(w, err)
		return
	}

	respond(w, result, http.StatusOK)

}
