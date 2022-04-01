package services

import (
	"context"
	"fmt"
	"github.com/sanity-io/litter"
	"log"
	. "social/internal/models"
	"strings"
)

//ToggleLikeOutput response model
type ToggleLikeOutput struct {
	Liked      bool `json:"liked,omitempty"`
	LikesCount int  `json:"likes_count,omitempty"`
}

//CreatePost adds new post to db and timeline
func (s *Service) CreatePost(
	ctx context.Context,
	content string,
	spoilerOf *string,
	nsfw bool,
) (TimelineItem, error) {
	var result TimelineItem
	userId, ok := ctx.Value(KeyAuthUserId).(int64)
	if !ok {
		return result, fmt.Errorf("unauthorized")
	}
	content = strings.TrimSpace(content)

	if content == "" || len([]rune(content)) > 480 {
		return result, fmt.Errorf("bad content")
	}

	if spoilerOf != nil {
		*spoilerOf = strings.TrimSpace(*spoilerOf)

		if *spoilerOf == "" || len([]rune(*spoilerOf)) > 64 {
			return result, fmt.Errorf("invalid spoiler")
		}
	}

	tx, err := s.Db.BeginTx(ctx, nil)

	if err != nil {
		return result, fmt.Errorf("cannot beging tx, %v", err)
	}

	defer tx.Rollback()

	query := `INSERT INTO posts (user_id, content, spoiler_of, nsfw) VALUES(@user_id,@content,@spoilerOf,@nsfw) returning id, created_at`

	query, args, err := queryBuilder(query, map[string]interface{}{
		"user_id":   userId,
		"content":   content,
		"spoilerOf": spoilerOf,
		"nsfw":      nsfw,
	})

	if err = tx.QueryRowContext(ctx, query, args...).Scan(&result.Post.Id, &result.Post.CreateAt); err != nil {
		return result, fmt.Errorf("cannot insert post to db, %v", err)
	}

	query = "insert into timeline (user_id, post_id) values (@user_id, @post_id) returning id"
	query, args, err = queryBuilder(query, map[string]interface{}{
		"user_id": userId,
		"post_id": result.Post.Id,
	})

	if err = tx.QueryRowContext(ctx, query, args...).Scan(&result.Id); err != nil {
		return result, fmt.Errorf("cannot insert timelineItem to db, %v", err)
	}

	tx.Commit()
	result.UserId = userId
	result.Post.Content = content
	result.Post.SpoilerOf = spoilerOf
	result.Post.NSFW = nsfw
	result.Post.UserId = userId
	result.Post.Mine = true
	result.PostId = result.Post.Id

	go func(p Post) {
		user, err := s.GetUserById(context.Background(), userId)
		if err != nil {
			log.Printf("cannot find user for post, %v", err)
		}
		result.Post.User = user
		result.Post.Mine = false

		postList, err := s.fanoutPost(result.Post)
		if err != nil {
			log.Printf("could not fanout posts, %v", err)
			return
		}
		for _, result = range postList {
			log.Println(litter.Sdump(result))
		}
	}(result.Post)

	return result, nil
}

// TogglePostLike add likes to post
func (s *Service) TogglePostLike(ctx context.Context, postId int64) (ToggleLikeOutput, error) {
	var result ToggleLikeOutput

	userId, ok := ctx.Value(KeyAuthUserId).(int64)
	if !ok {
		return result, fmt.Errorf("unauthorized")
	}

	tx, err := s.Db.BeginTx(ctx, nil)

	if err != nil {
		return result, fmt.Errorf("cannot beging tx, %v", err)
	}

	defer tx.Rollback()

	query := `select exists (
				select 1 from post_likes where user_id = @userId and post_id = @postId
				)`
	query, args, err := queryBuilder(query, map[string]interface{}{
		"userId": userId,
		"postId": postId,
	})

	if err = tx.QueryRow(query, args...).Scan(&result.Liked); err != nil {
		return result, fmt.Errorf("could not select posts like existance, %v", err)
	}

	if result.Liked {
		query = `delete from post_likes where user_id = $userId and post_id = @postId`
		query, args, err := queryBuilder(query, map[string]interface{}{
			"userId": userId,
			"postId": postId,
		})
		if err != nil {
			return result, fmt.Errorf("cannot generate query , %v", err)
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return result, fmt.Errorf("could not remove like from post, %v", err)
		}

		query = "update posts set likes_count = likes_count -1 where id = $1 returning likes_count"
		if err = tx.QueryRowContext(ctx, query, postId).Scan(&result.LikesCount); err != nil {
			return result, fmt.Errorf("could not update post count, %v", err)
		}
	} else {
		query = "insert into post_likes (user_id, post_id) values ($1,$2)"
		_, err = tx.ExecContext(ctx, query, userId, postId)
		if err != nil {
			return result, fmt.Errorf("could not insert like for post, %v", err)
		}
		query = "update posts set likes_count = likes_count + 1 where id = $1 returning likes_count"
		if err = tx.QueryRowContext(ctx, query, postId).Scan(&result.LikesCount); err != nil {
			return result, fmt.Errorf("could not update post count, %v", err)
		}

		if err = tx.Commit(); err != nil {
			return result, fmt.Errorf("could not commit tx, %v", err)
		}

		result.Liked = !result.Liked
	}

	return result, nil
}

func (s *Service) GetMyPosts(ctx context.Context) ([]Post, error) {
	var postList []Post
	userId, ok := ctx.Value(KeyAuthUserId).(int64)
	if !ok {
		return nil, fmt.Errorf("unauthorized")
	}

	query := ` select p.id, content, nsfw, spoiler_of, user_id, created_at,u.username, u.avatar_url from posts p
 	left join users u on u.id = p.user_id
 	where user_id = @userId order by created_at desc limit 10 offset 2`
	query, args, err := queryBuilder(query, map[string]interface{}{
		"userId": userId,
	})
	if err != nil {
		return nil, fmt.Errorf("could not generate query, %v", err)
	}

	rows, err := s.Db.QueryContext(ctx, query, args...)

	for rows.Next() {
		var item Post
		rows.Scan(&item.Id, &item.Content, &item.NSFW, &item.SpoilerOf, &item.UserId,
			&item.CreateAt, &item.User.Username, &item.User.AvatarUrl)

		if item.UserId == userId {
			item.Mine = true
		}
		postList = append(postList, item)
	}

	return postList, nil
}

//GetPosts fetch all posts (implement pagination later)
func (s *Service) GetPosts(ctx context.Context) ([]Post, error) {
	var postList []Post
	userId, ok := ctx.Value(KeyAuthUserId).(int64)
	if !ok {
		return nil, fmt.Errorf("unauthorized")
	}

	query := ` select p.id, content, nsfw, spoiler_of, user_id, created_at,u.username, u.avatar_url from posts p
 	left join users u on u.id = p.user_id order by created_at desc limit 10`
	query, args, err := queryBuilder(query, map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("could not generate query, %v", err)
	}

	rows, err := s.Db.QueryContext(ctx, query, args...)

	for rows.Next() {
		var item Post
		rows.Scan(&item.Id, &item.Content, &item.NSFW, &item.SpoilerOf, &item.UserId,
			&item.CreateAt, &item.User.Username, &item.User.AvatarUrl)

		if item.UserId == userId {
			item.Mine = true
		}
		postList = append(postList, item)
	}

	return postList, nil
}

// GetPostsByUserId fetch posts for specific user
func (s *Service) GetPostsByUserId(ctx context.Context, userId int64) ([]Post, error) {
	var postList []Post
	_, ok := ctx.Value(KeyAuthUserId).(int64)
	if !ok {
		return nil, fmt.Errorf("unauthorized")
	}

	query := ` select p.id, content, nsfw, spoiler_of, user_id, created_at,u.username, u.avatar_url from posts p
 		left join users u on u.id = p.user_id 
 		where p.user_id = @user_id
 		order by created_at desc limit 10`
	query, args, err := queryBuilder(query, map[string]interface{}{
		"userId": userId,
	})
	if err != nil {
		return nil, fmt.Errorf("could not generate query, %v", err)
	}

	rows, err := s.Db.QueryContext(ctx, query, args...)

	for rows.Next() {
		var item Post
		rows.Scan(&item.Id, &item.Content, &item.NSFW, &item.SpoilerOf, &item.UserId,
			&item.CreateAt, &item.User.Username, &item.User.AvatarUrl)

		if item.UserId == userId {
			item.Mine = true
		}
		postList = append(postList, item)
	}

	return postList, nil
}

// GetPostById fetch single post from db
func (s *Service) GetPostById(ctx context.Context, postId int64) (Post, error) {
	var post Post
	userId, ok := ctx.Value(KeyAuthUserId).(int64)
	if !ok {
		return post, fmt.Errorf("unauthorized")
	}

	query := ` select p.id, content, nsfw, spoiler_of, user_id, created_at,u.username, u.avatar_url from posts p
 	left join users u on u.id = p.user_id where p.id = @postId`
	query, args, err := queryBuilder(query, map[string]interface{}{
		"postId": postId,
	})
	if err != nil {
		return post, fmt.Errorf("could not generate query, %v", err)
	}

	if err = s.Db.QueryRowContext(ctx, query, args...).Scan(&post.Id, &post.Content, &post.NSFW, &post.SpoilerOf, &post.UserId,
		&post.CreateAt, &post.User.Username, &post.User.AvatarUrl); err != nil {
		return post, fmt.Errorf("could not fetch post from db")
	}

	if post.UserId == userId {
		post.Mine = true
	}

	return post, nil
}

// Private methods
func (s *Service) fanoutPost(p Post) ([]TimelineItem, error) {
	query := "insert into timeline (user_id,post_id) " +
		"Select follower_id, $1 from follows where followee_id = $2 " +
		"returning id, user_id"
	rows, err := s.Db.Query(query, p.Id, p.UserId)
	if err != nil {
		return nil, fmt.Errorf("cannot insert timeline: %v", err)
	}

	defer rows.Close()

	var itemList []TimelineItem
	for rows.Next() {
		var item TimelineItem
		if err = rows.Scan(&item.Id, &item.PostId); err != nil {
			return nil, err
		}
		item.PostId = p.Id
		item.Post = p
		itemList = append(itemList, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("cannot iterate list of posts, %v", err)
	}

	return itemList, err
}
