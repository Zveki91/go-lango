package models

import "time"

type Comment struct {
	Id         int64     `json:"id"`
	UserId     int64     `json:"userId"`
	PostId     int64     `json:"postId"`
	Content    string    `json:"content"`
	LikesCount int       `json:"likes_count"`
	CreatedAt  time.Time `json:"createdAt"`
	User       *User     `json:"user"`
	Mine       bool      `json:"mine"`
	Liked      bool      `json:"liked"`
}
