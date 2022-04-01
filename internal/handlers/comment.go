package handlers

import (
	"encoding/json"
	"github.com/matryer/way"
	"net/http"
	"strconv"
)

type CreateComment struct {
	Content string `json:"content"`
}

func (h *Handler) createComment(w http.ResponseWriter, r *http.Request) {
	var input CreateComment
	defer r.Body.Close()
	ctx := r.Context()
	_ = json.NewDecoder(r.Body).Decode(&input)
	id, err := strconv.ParseInt(way.Param(ctx, "id"), 10, 64)
	if err != nil {
		respondError(w, err)
		return
	}

	result, err := h.CreateComment(ctx, input.Content, id)
	if err != nil {
		respondError(w, err)
		return
	}

	respond(w, result, http.StatusCreated)

}
