package models

import "time"

type Post struct {
	Id        int64     `json:"id"`
	UserId    int64     `json:"userId"`
	Content   string    `json:"content"`
	SpoilerOf *string   `json:"spoilerOf"`
	CreateAt  time.Time `json:"createAt"`
	NSFW      bool      `json:"nsfw"`
	User      User      `json:"user,omitempty"`
	Mine      bool      `json:"mine"`
}
