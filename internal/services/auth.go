package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	. "social/internal/models"
	"strconv"
	"time"
)

const (
	// TokenLifeSpan token expiration time
	TokenLifeSpan = time.Hour * 24 * 14
	// KeyAuthUserId userId in http context
	KeyAuthUserId = "auth_user_id"
)

type LoginOutput struct {
	Token     string
	ExpiresAt time.Time
	AuthUser  User
}

func (s *Service) Login(ctx context.Context, email string) (LoginOutput, error) {
	var out LoginOutput

	query := fmt.Sprintf("Select id,username from public.users where email = '%s'", email)
	err := s.Db.QueryRowContext(ctx, query).Scan(&out.AuthUser.Id, &out.AuthUser.Username)

	if err == sql.ErrNoRows {
		return out, errors.New("record not found")
	}

	if err != nil {
		return out, fmt.Errorf("cannot find user")
	}

	out.Token, err = s.Codec.EncodeToString(strconv.FormatInt(out.AuthUser.Id, 10))
	if err != nil {
		return out, fmt.Errorf("cannot generate token")
	}
	out.ExpiresAt = time.Now().Add(TokenLifeSpan)

	return out, nil
}

func (s *Service) AuthUser(ctx context.Context) (User, error) {
	var user User
	userId, ok := ctx.Value(KeyAuthUserId).(int64)
	if !ok {
		return user, fmt.Errorf("no user in session")
	}

	query := fmt.Sprintf("Select username from users where id = %d", userId)
	err := s.Db.QueryRowContext(ctx, query).Scan(&user.Username)

	if err == sql.ErrNoRows {
		return user, fmt.Errorf("record not found")
	}

	if err != nil {
		return user, fmt.Errorf("cannot find user")
	}

	user.Id = userId
	return user, nil

}

func (s *Service) Authorized(token string) (int64, error) {
	str, err := s.Codec.DecodeToString(token)
	if err != nil {
		return 0, fmt.Errorf("could not decode token: %v", err)
	}

	id, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse auth user id from token: %v", err)
	}

	return id, err
}
