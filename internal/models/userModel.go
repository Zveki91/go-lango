package models

// User model
type User struct {
	Id        int64   `json:"id,omitempty"`
	Username  string  `json:"username"`
	AvatarUrl *string `json:"avatarUrl"`
}
