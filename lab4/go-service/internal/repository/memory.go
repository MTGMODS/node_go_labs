package repository

import (
	"sort"
	"strings"
	"sync"

	"node-go-labs/lab4/go-service/internal/model"
)

type InMemoryUserRepository struct {
	mu     sync.RWMutex
	users  map[int]model.User
	nextID int
}

var _ UserRepository = (*InMemoryUserRepository)(nil)

func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{users: make(map[int]model.User), nextID: 1}
}

func (r *InMemoryUserRepository) List() ([]model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	users := make([]model.User, 0, len(r.users))
	for _, user := range r.users {
		users = append(users, user)
	}
	sort.Slice(users, func(i, j int) bool { return users[i].ID < users[j].ID })
	return users, nil
}

func (r *InMemoryUserRepository) FindByID(id int) (model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user, ok := r.users[id]
	if !ok {
		return model.User{}, ErrNotFound
	}
	return user, nil
}

func (r *InMemoryUserRepository) FindByEmail(email string) (model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, user := range r.users {
		if strings.EqualFold(user.Email, email) {
			return user, nil
		}
	}
	return model.User{}, ErrNotFound
}

func (r *InMemoryUserRepository) Create(user model.User) (model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	user.ID = r.nextID
	r.nextID++
	r.users[user.ID] = user
	return user, nil
}

func (r *InMemoryUserRepository) Update(user model.User) (model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.users[user.ID]; !ok {
		return model.User{}, ErrNotFound
	}
	r.users[user.ID] = user
	return user, nil
}

func (r *InMemoryUserRepository) Delete(id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.users[id]; !ok {
		return ErrNotFound
	}
	delete(r.users, id)
	return nil
}
