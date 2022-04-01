package services

import (
	"context"
	"fmt"
	. "social/internal/models"
)

func (s *Service) CreateComment(ctx context.Context, content string, postId int64) (Comment, error) {
	var result Comment
	userId, ok := ctx.Value(KeyAuthUserId).(int64)
	if !ok {
		return result, fmt.Errorf("unauthorized")
	}

	query := `insert into comments(user_id,post_id,content) values(@userId,@postId,@content) returning id`
	query, args, err := queryBuilder(query, map[string]interface{}{
		"userId":  userId,
		"postId":  postId,
		"content": content,
	})

	if err = s.Db.QueryRowContext(ctx, query, args...).Scan(&result.Id); err != nil {
		return result, fmt.Errorf("cannot insert comment")
	}
	user, err := s.GetUserById(ctx, userId)
	if err != nil {
		return result, fmt.Errorf("cannot find user, %v", err)
	}

	result.Liked = false
	result.PostId = postId
	result.Content = content
	result.UserId = userId
	result.User = &User{
		Id:        userId,
		Username:  user.Username,
		AvatarUrl: user.AvatarUrl,
	}
	result.LikesCount = 0
	result.Mine = true

	return result, nil

}
