package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"social/internal/services"
	"strings"
)

type loginInput struct {
	Email string
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var in loginInput
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	out, err := h.Login(r.Context(), in.Email)
	if err != nil {
		respondError(w, err)
		return
	}

	respond(w, out, http.StatusOK)
}

func (h *Handler) authUser(w http.ResponseWriter, r *http.Request) {
	user, err := h.AuthUser(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	respond(w, user, http.StatusOK)

}

func (h *Handler) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if !strings.HasPrefix(token, "Bearer") {
			next.ServeHTTP(w, r)
			return
		}
		result := token[7:]
		userId, err := h.Authorized(result)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		ctx := r.Context()
		ctx = context.WithValue(ctx, services.KeyAuthUserId, userId)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
