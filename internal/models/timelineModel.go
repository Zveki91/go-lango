package models

type TimelineItem struct {
	Id     int64 `json:"id"`
	UserId int64 `json:"userId"`
	PostId int64 `json:"postId"`
	Post   Post  `json:"post"`
}
