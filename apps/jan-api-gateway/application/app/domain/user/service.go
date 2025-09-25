package user

import (
	"fmt"

	"golang.org/x/net/context"
	"menlo.ai/jan-api-gateway/app/infrastructure/cache"
	"menlo.ai/jan-api-gateway/app/utils/idgen"
	"menlo.ai/jan-api-gateway/app/utils/logger"
)

type UserService struct {
	userrepo UserRepository
	cache    cache.CacheService
}

func NewService(userrepo UserRepository, cacheService cache.CacheService) *UserService {
	return &UserService{
		userrepo: userrepo,
		cache:    cacheService,
	}
}

func (s *UserService) RegisterUser(ctx context.Context, user *User) (*User, error) {
	publicId, err := s.generatePublicID()
	if err != nil {
		return nil, err
	}
	user.PublicID = publicId
	if err := s.userrepo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) UpdateUser(ctx context.Context, user *User) (*User, error) {
	// Use distributed lock to prevent race conditions
	lockKey := fmt.Sprintf(cache.UserLockKey, user.PublicID)

	var result *User
	var updateErr error

	// Execute with lock using go-redsync
	err := cache.WithLock(s.cache, lockKey, func() error {
		// Update database
		if err := s.userrepo.Update(ctx, user); err != nil {
			updateErr = err
			return err
		}

		// Invalidate cache for this user
		if user.PublicID != "" {
			cacheKey := fmt.Sprintf(cache.UserByPublicIDKey, user.PublicID)
			if cacheErr := s.cache.Unlink(ctx, cacheKey); cacheErr != nil {
				// Log cache error but don't fail the request
				logger.GetLogger().Errorf("failed to invalidate cache for user %s: %v", user.PublicID, cacheErr)
			}
		}

		result = user
		return nil
	}, cache.UserLockTTL)

	if err != nil {
		logger.GetLogger().Warnf("Failed to acquire lock for user %s update: %v", user.PublicID, err)
		// Still update the database even if we can't acquire lock
		if err := s.userrepo.Update(ctx, user); err != nil {
			return nil, err
		}
		return user, nil
	}

	if updateErr != nil {
		return nil, updateErr
	}

	return result, nil
}

func (s *UserService) FindByEmail(ctx context.Context, email string) (*User, error) {
	users, err := s.userrepo.FindByFilter(ctx, UserFilter{
		Email: &email,
	}, nil)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, nil
	}
	if len(users) != 1 {
		return nil, fmt.Errorf("invalid email")
	}
	return users[0], nil
}

func (s *UserService) FindByFilter(ctx context.Context, filter UserFilter) ([]*User, error) {
	return s.userrepo.FindByFilter(ctx, filter, nil)
}

func (s *UserService) FindByID(ctx context.Context, id uint) (*User, error) {
	return s.userrepo.FindByID(ctx, id)
}

func (s *UserService) FindByPublicID(ctx context.Context, publicID string) (*User, error) {
	// Create cache key
	cacheKey := fmt.Sprintf(cache.UserByPublicIDKey, publicID)

	// Try to get from cache first
	var cachedUser *User
	err := s.cache.Get(ctx, cacheKey, &cachedUser)
	if err == nil && cachedUser != nil {
		return cachedUser, nil
	}

	// Cache miss or error - fetch from database
	userEntities, err := s.userrepo.FindByFilter(ctx, UserFilter{PublicID: &publicID}, nil)
	if err != nil {
		return nil, err
	}
	if len(userEntities) != 1 {
		return nil, fmt.Errorf("user does not exist")
	}

	user := userEntities[0]

	// Cache the result for future requests
	if cacheErr := s.cache.Set(ctx, cacheKey, user, cache.UserCacheTTL); cacheErr != nil {
		// Log cache error but don't fail the request
		// TODO: Add proper logging here
	}

	return user, nil
}

func (s *UserService) generatePublicID() (string, error) {
	return idgen.GenerateSecureID("user", 24)
}
