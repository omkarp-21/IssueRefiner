package users

import (
	"context"
	"errors"
	"issue-refiner/internal/github"

	"github.com/jackc/pgx/v5"
)

type UserService struct {
	UserRepository *UserRepository
}

func NewUserService(userRepository *UserRepository) UserService {
	return UserService{UserRepository: userRepository}
}

func (userService *UserService) GetOrCreateFromGitHub(ctx context.Context, githubUser *github.GitHubUser) (*User, error) {
	var (
		user *User
		err  error
	)
	user, err = userService.UserRepository.FindByGitHubID(ctx, githubUser.ID)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			user, err = userService.UserRepository.Create(ctx, githubUser)
		default:
			return nil, err
		}
	}
	return user, nil
}
