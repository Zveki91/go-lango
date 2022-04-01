package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/disintegration/imaging"
	gonanoid "github.com/matoous/go-nanoid"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"os"
	"path"
	"regexp"
	. "social/internal/models"
	"strings"
)

const (
	MaxAvatarBytes = 5 << 20
)

var (
	rxEmail    = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	rxUsername = regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9_-]{0,17}$")
	avatarDir  = path.Join("web", "static", "img", "avatars")
)

// ToggleFollowOutput output dto
type ToggleFollowOutput struct {
	Following      bool
	FollowersCount int
}

// CreateUser creates new user
func (s *Service) CreateUser(ctx context.Context, email, username string) error {
	email = strings.TrimSpace(email)

	if !rxEmail.MatchString(email) {
		return errors.New("bad email address")
	}

	username = strings.TrimSpace(username)
	if !rxUsername.MatchString(username) {
		return errors.New("bad username")
	}

	query := "Insert into public.users(email,username) values($1,$2)"
	_, err := s.Db.ExecContext(ctx, query, email, username)

	if err != nil {
		return fmt.Errorf("could not insert user")
	}

	return nil
}

// GetUserById fetch single user info
func (s *Service) GetUserById(ctx context.Context, userId int64) (User, error) {
	var user User
	query := fmt.Sprintf("Select username, avatar_url from users where id = %d", userId)
	err := s.Db.QueryRowContext(ctx, query).Scan(&user.Username, &user.AvatarUrl)

	if err == sql.ErrNoRows {
		return user, fmt.Errorf("record not found")
	}

	if err != nil {
		return user, fmt.Errorf("cannot find user")
	}

	user.Id = userId
	return user, nil

}

// ToggleFollow between wto users
func (s *Service) ToggleFollow(ctx context.Context, username string) (ToggleFollowOutput, error) {
	var out ToggleFollowOutput
	followerId, ok := ctx.Value(KeyAuthUserId).(int64)
	if !ok {
		return out, fmt.Errorf("user not authenticated")
	}

	username = strings.TrimSpace(username)

	if !rxUsername.MatchString(username) {
		return out, fmt.Errorf("invalid username")
	}

	var followeeId int64
	tx, err := s.Db.BeginTx(ctx, nil)
	if err != nil {
		return out, fmt.Errorf("could not begin tx: %v", err)
	}
	defer tx.Rollback()
	query := fmt.Sprintf("Select id from users where username = $1")
	if err = tx.QueryRowContext(ctx, query, username).Scan(&followeeId); err != nil {
		return out, fmt.Errorf("could not find user")
	}

	if followeeId == followerId {
		return out, fmt.Errorf("you cannot follow yourself")
	}

	query = "Select exists (select 1 from follows where follower_id = $1 and followee_id = $2)"
	if err = tx.QueryRowContext(ctx, query, followerId, followeeId).Scan(&out.Following); err != nil {
		return out, fmt.Errorf("could not query select existance of follow: %v", err)
	}

	if out.Following {
		query = "Delete from follows where follower_id = $1 and followee_id == $2"
		if _, err = tx.ExecContext(ctx, query, followerId, followeeId); err != nil {
			return out, fmt.Errorf("could not delete follow: %v", err)
		}
		query = "Update users set followees_count = followees_count - 1 where id = $1"
		if _, err = tx.ExecContext(ctx, query, followerId); err != nil {
			return out, fmt.Errorf("could not update follower followees count (-): %v", err)
		}
		query = "update users set followers_count = followers_count - 1 where id = $1 returning followers_count"
		if err = tx.QueryRowContext(ctx, query, followeeId).Scan(&out.FollowersCount); err != nil {
			return out, fmt.Errorf("could not update followee followers count(-): %v", err)
		}
	} else {
		query = "insert into follows (follower_id, followee_id) values($1,$2)"
		if _, err = tx.ExecContext(ctx, query, followerId, followeeId); err != nil {
			return out, fmt.Errorf("cannot add new row to follows: %v", err)
		}
		query = "update users set followees_count = followees_count + 1 where id = $1"
		if _, err = tx.ExecContext(ctx, query, followerId); err != nil {
			return out, fmt.Errorf("error updataing number of followees (+) : %v", err)
		}

		query = "update users set followers_count = followers_count + 1 where id = $1 returning followers_count"
		if err = tx.QueryRowContext(ctx, query, followeeId).Scan(&out.FollowersCount); err != nil {
			return out, fmt.Errorf("error updating number of followers(+): %v", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return out, fmt.Errorf("error while commiting tx, %v", err)
	}

	out.Following = !out.Following

	if out.Following {
		// TODO: notify followee
	}

	return out, nil

}

func (s *Service) GetUsers(ctx context.Context, search string, first int, after string) ([]UserProfile, error) {
	search = strings.TrimSpace(search)
	after = strings.TrimSpace(after)
	first = normalizePageSize(first)
	uid, auth := ctx.Value(KeyAuthUserId).(int64)
	query, args, err := queryBuilder(`SELECT id, email,avatar_url, username, followers_count, followees_count
		{{ if .auth }}
		, followers.follower_id IS NOT NULL AS following
		, followees.followee_id IS NOT NULL AS followeed
		{{ end }}
		FROM users
		{{ if .auth }}
		LEFT JOIN follows AS followers
			ON followers.follower_id = @uid AND followers.followee_id = users.id
		LEFT JOIN follows AS followees
			ON followees.follower_id = users.id AND followees.followee_id = @uid
		{{ end }}
		{{ if or .search .afterUsername }}WHERE{{ end }}
		{{ if .search }}username ILIKE '%' || @search || '%'{{ end }}
		{{ if and .search .after }}AND{{ end }}
		{{ if .after }}username > @after{{ end }}
		ORDER BY username ASC
		LIMIT @first`, map[string]interface{}{
		"auth":   auth,
		"uid":    uid,
		"search": search,
		"first":  first,
		"after":  after,
	})
	if err != nil {
		return nil, fmt.Errorf("could not build users query , %v", err)
	}

	log.Printf("users query : %s\n args : %v\n ", query, args)

	rows, err := s.Db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not execute query : %v", err)
	}

	defer rows.Close()

	userProfileList := make([]UserProfile, 0, first)

	for rows.Next() {
		var profile UserProfile
		dest := []interface{}{&profile.Id, &profile.Email, &profile.AvatarUrl, &profile.Username, &profile.FolloweesCount, &profile.FollowersCount}
		if auth {
			dest = append(dest, &profile.Followed, &profile.Following)
		}
		if err = rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("could not parse results, %v", err)
		}

		profile.Me = auth && uid == profile.Id
		if !profile.Me {
			profile.Id = 0
			profile.Email = ""
		}
		userProfileList = append(userProfileList, profile)
	}

	return userProfileList, nil
}

// GetUserProfile fetch user profile from db
func (s *Service) GetUserProfile(ctx context.Context, username string) (UserProfile, error) {
	var userProfile UserProfile

	username = strings.TrimSpace(username)

	if !rxUsername.MatchString(username) {
		return userProfile, fmt.Errorf("invalid username")
	}

	uid, auth := ctx.Value(KeyAuthUserId).(int64)
	args := []interface{}{username}
	dest := []interface{}{&userProfile.Id, &userProfile.Email, &userProfile.Username, &userProfile.FollowersCount, &userProfile.AvatarUrl, &userProfile.FolloweesCount}

	query := "select id, email, username,avatar_url, followers_count, followees_count"
	if auth {
		query += ", " + " followers.follower_id is not null as following," +
			" followees.followee_id is not null as followeed"
		dest = append(dest, &userProfile.Following, &userProfile.Followed)
	}
	query += " from users"
	if auth {
		query += " left join follows as followers on followers.follower_id = $2 and followers.followee_id = users.id " +
			" left join follows as followees on followees.follower_id = users.id and followees.followee_id = $2"
		args = append(args, uid)

	}
	query += " where username = $1"

	err := s.Db.QueryRowContext(ctx, query, args...).Scan(dest...)

	if err != nil {
		return userProfile, fmt.Errorf("error while fetching user profile, %v", err)
	}

	userProfile.Username = username
	userProfile.Me = auth && uid == userProfile.Id
	if !userProfile.Me {
		userProfile.Id = 0
		userProfile.Email = ""
	}

	return userProfile, nil
}

//GetFollowers fetch followers from db
func (s *Service) GetFollowers(ctx context.Context, username string, first int, after string) ([]UserProfile, error) {
	username = strings.TrimSpace(username)
	after = strings.TrimSpace(after)
	first = normalizePageSize(first)
	uid, auth := ctx.Value(KeyAuthUserId).(int64)
	query, args, err := queryBuilder(`SELECT users.id,users.avatarUrl, users.email, users.username, users.followers_count, users.followees_count
		{{ if .auth }}
		, followers.follower_id IS NOT NULL AS following
		, followees.followee_id IS NOT NULL AS followeed
		{{ end }}
		from follows 
		inner join users on follows.follower_id = users.id
		{{ if .auth }}
		LEFT JOIN follows AS followers
			ON followers.follower_id = @uid AND followers.followee_id = users.id
		LEFT JOIN follows AS followees
			ON followees.follower_id = users.id AND followees.followee_id = @uid
		{{ end }}
		where follows.followee_id = (select id from users where username = @username)
		{{ if .after }}username > @after{{ end }}
		ORDER BY username ASC
		LIMIT @first`, map[string]interface{}{
		"auth":     auth,
		"uid":      uid,
		"username": username,
		"first":    first,
		"after":    after,
	})
	if err != nil {
		return nil, fmt.Errorf("could not build users query , %v", err)
	}

	log.Printf("followers query : %s\n args : %v\n ", query, args)

	rows, err := s.Db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not execute query : %v", err)
	}

	defer rows.Close()

	userProfileList := make([]UserProfile, 0, first)

	for rows.Next() {
		var profile UserProfile
		dest := []interface{}{&profile.Id, &profile.Email, &profile.Username, &profile.FolloweesCount, &profile.FollowersCount}
		if auth {
			dest = append(dest, &profile.Followed, &profile.Following)
		}
		if err = rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("could not parse results, %v", err)
		}

		profile.Me = auth && uid == profile.Id
		if !profile.Me {
			profile.Id = 0
			profile.Email = ""
		}
		userProfileList = append(userProfileList, profile)
	}

	return userProfileList, nil
}

// GetFollowees fetch followees from db
func (s *Service) GetFollowees(ctx context.Context, username string, first int, after string) ([]UserProfile, error) {
	username = strings.TrimSpace(username)
	after = strings.TrimSpace(after)
	first = normalizePageSize(first)
	uid, auth := ctx.Value(KeyAuthUserId).(int64)
	query, args, err := queryBuilder(`SELECT users.id, users.email,users.avatarUrl, users.username, users.followers_count, users.followees_count
		{{ if .auth }}
		, followers.follower_id IS NOT NULL AS following
		, followees.followee_id IS NOT NULL AS followeed
		{{ end }}
		from follows 
		inner join users on follows.followee_id = users.id
		{{ if .auth }}
		LEFT JOIN follows AS followers
			ON followers.follower_id = @uid AND followers.followee_id = users.id
		LEFT JOIN follows AS followees
			ON followees.follower_id = users.id AND followees.followee_id = @uid
		{{ end }}
		where follows.follower_id = (select id from users where username = @username)
		{{ if .after }}username > @after{{ end }}
		ORDER BY username ASC
		LIMIT @first`, map[string]interface{}{
		"auth":     auth,
		"uid":      uid,
		"username": username,
		"first":    first,
		"after":    after,
	})
	if err != nil {
		return nil, fmt.Errorf("could not build users query , %v", err)
	}

	log.Printf("followers query : %s\n args : %v\n ", query, args)

	rows, err := s.Db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not execute query : %v", err)
	}

	defer rows.Close()

	userProfileList := make([]UserProfile, 0, first)

	for rows.Next() {
		var profile UserProfile
		dest := []interface{}{&profile.Id, &profile.Email, &profile.Username, &profile.FolloweesCount, &profile.FollowersCount}
		if auth {
			dest = append(dest, &profile.Followed, &profile.Following)
		}
		if err = rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("could not parse results, %v", err)
		}

		profile.Me = auth && uid == profile.Id
		if !profile.Me {
			profile.Id = 0
			profile.Email = ""
		}
		userProfileList = append(userProfileList, profile)
	}

	return userProfileList, nil
}

// UpdateAvatar upload anad update avatar image
func (s *Service) UpdateAvatar(ctx context.Context, r io.Reader) (string, error) {
	userId, ok := ctx.Value(KeyAuthUserId).(int64)

	if !ok {
		return "", fmt.Errorf("unauthorized")
	}

	r = io.LimitReader(r, MaxAvatarBytes)
	img, format, err := image.Decode(r)
	if err != nil {
		return "", fmt.Errorf("unable to decode image, %v", err)
	}

	if format != "png" && format != "jpeg" {
		return "", fmt.Errorf("unsupported image format")
	}

	avatar, err := gonanoid.Nanoid()
	if err != nil {
		return "", fmt.Errorf("unable to generate guid for avatar , %v", err)
	}

	if format == "png" {
		avatar += ".png"
	} else {
		avatar += ".jpg"
	}

	avatarPath := path.Join(avatarDir, avatar)

	f, err := os.Create(avatarPath)
	if err != nil {
		return "", fmt.Errorf("could not create avatar file, %v", err)
	}

	defer f.Close()

	img = imaging.Fill(img, 400, 400, imaging.Center, imaging.CatmullRom)
	if format == "png" {
		err = png.Encode(f, img)
	} else {
		err = jpeg.Encode(f, img, nil)
	}

	if err != nil {
		return "", fmt.Errorf("could not encode image: %v", err)
	}

	query := `update users set avatar_url = @avatarUrl where id = @id returning (select avatar_url from users where id = @id) as old_avatar`
	query, args, err := queryBuilder(query, map[string]interface{}{
		"id":        userId,
		"avatarUrl": avatar,
	})

	if err != nil {
		defer os.Remove(path.Join(avatarDir, avatar))
		fmt.Errorf("error build query for updating avatar: %v", err)
		return "", err
	}
	var oldAvatar sql.NullString
	if err = s.Db.QueryRowContext(ctx, query, args...).Scan(&oldAvatar); err != nil {
		defer os.Remove(avatarPath)
		fmt.Errorf("error build query for updating avatar: %v", err)
		return "", err
	}
	if oldAvatar.Valid {
		defer os.Remove(path.Join(avatarDir, oldAvatar.String))
	}

	return s.Origin + "/img/avatars/" + avatar, nil

}
