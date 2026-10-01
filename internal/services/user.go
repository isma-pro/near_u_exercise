package services

import (
	"fmt"

	"github.com/ismaelucky94/near_u_exercise/internal/models"
)

// UserService contains business logic for managing users.
type UserService struct {
	users map[string]*models.User
}

// NewUserService returns a new UserService.
func NewUserService() *UserService {
	return &UserService{
		users: make(map[string]*models.User),
	}
}

// List returns all stored users.
func (s *UserService) List() []*models.User {
	result := make([]*models.User, 0, len(s.users))
	for _, u := range s.users {
		result = append(result, u)
	}
	return result
}

// Get returns a user by ID.
func (s *UserService) Get(id string) (*models.User, error) {
	user, ok := s.users[id]
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

// Create stores a new user.
func (s *UserService) Create(user *models.User) {
	s.users[user.ID] = user
}

// Update replaces an existing user.
func (s *UserService) Update(id string, user *models.User) (*models.User, error) {
	if _, ok := s.users[id]; !ok {
		return nil, fmt.Errorf("user not found")
	}
	user.ID = id
	s.users[id] = user
	return user, nil
}

// Delete removes a user by ID.
func (s *UserService) Delete(id string) error {
	if _, ok := s.users[id]; !ok {
		return fmt.Errorf("user not found")
	}
	delete(s.users, id)
	return nil
}
