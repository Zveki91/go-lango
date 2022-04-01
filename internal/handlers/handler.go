package handlers

import (
	"github.com/matryer/way"
	"net/http"
	"social/internal/services"
)

type Handler struct {
	*services.Service
}

// New creates new HTTP handler
func New(s *services.Service) http.Handler {
	h := &Handler{s}

	api := way.NewRouter()
	// user routes
	api.HandleFunc("GET", "/users/:username/profile", h.getUserProfile)
	api.HandleFunc("GET", "/users/followers", h.getFollowers)
	api.HandleFunc("GET", "/users/follows", h.getFollows)
	api.HandleFunc("GET", "/users", h.getUserProfiles)
	api.HandleFunc("POST", "/users/:username/toggle_follow", h.toggleFollow)
	api.HandleFunc("POST", "/users", h.createUser)

	// Auth routes
	api.HandleFunc("GET", "/auth_user", h.authUser)
	api.HandleFunc("POST", "/login", h.login)

	// Posts routes
	api.HandleFunc("POST", "/posts", h.createPost)
	api.HandleFunc("GET", "/posts", h.getPosts)
	api.HandleFunc("GET", "/posts/:postId", h.getPostById)
	api.HandleFunc("GET", "/posts/users/:id", h.getPostsForUser)
	api.HandleFunc("GET", "/posts/me", h.getMyPosts)
	api.HandleFunc("Post", "/posts/:postId/like", h.togglePostLike)

	// Comment routes

	api.HandleFunc("POST", "/comment/:id", h.createComment)

	// Patch Methods
	api.HandleFunc("PATCH", "/auth_user/avatar", h.updateAvatar)

	r := way.NewRouter()
	r.Handle("*", "/api...", http.StripPrefix("/api", h.withAuth(api)))

	return r
}
