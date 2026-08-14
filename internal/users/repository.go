package users

import (
	"context"
	"errors"
	"fmt"
	"issue-refiner/internal/github"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCreateUser = errors.New("failed to insert new user")
)

type UserRepository struct {
	dbPool *pgxpool.Pool
}

func NewUserRepository(dbPool *pgxpool.Pool) UserRepository {
	return UserRepository{dbPool: dbPool}
}

func (userRepository *UserRepository) FindByGitHubID(ctx context.Context, githubId int64) (*User, error) {
	var user User
	err := userRepository.dbPool.QueryRow(
		ctx,
		"SELECT id, github_id, username, display_name, avatar_url, email, created_at, updated_at from users WHERE github_id=$1",
		githubId,
	).Scan(&user.ID,
		&user.GitHubID,
		&user.Username,
		&user.DisplayName,
		&user.AvatarURL,
		&user.Email,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	return &user, err
}

func (userRepository *UserRepository) Create(ctx context.Context, githubUser *github.GitHubUser) (*User, error) {
	newID := uuid.New()
	currentTime := time.Now()

	user := User{
		ID:          newID,
		GitHubID:    githubUser.ID,
		Username:    githubUser.Login,
		DisplayName: githubUser.Name,
		Email:       &githubUser.Email,
		AvatarURL:   githubUser.AvatarURL,
		CreatedAt:   currentTime,
		UpdatedAt:   currentTime,
	}

	query := `
		INSERT INTO users (
		id, 
		github_id, 
		username, 
		display_name, 
		email,
		avatar_url,
		created_at,
		updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := userRepository.dbPool.Exec(ctx, query,
		user.ID,
		user.GitHubID,
		user.Username,
		user.DisplayName,
		user.Email,
		user.AvatarURL,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateUser, err)
	}
	return &user, nil
}
