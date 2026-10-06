package users

import (
	"context"
	"issue-refiner/internal/github"
)

type UserService struct {
	UserRepository *UserRepository
}

func NewUserService(userRepository *UserRepository) UserService {
	return UserService{UserRepository: userRepository}
}

func (userService *UserService) GetOrCreateFromGitHub(ctx context.Context, githubUser *github.GitHubUser) (*User, error) {
	return userService.UserRepository.Create(ctx, githubUser)
}
