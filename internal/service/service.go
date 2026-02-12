package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Kerhoff/TodoboT/internal/models"
	"github.com/Kerhoff/TodoboT/internal/repository"
	"github.com/sirupsen/logrus"
)

// Service is the central business logic layer that holds all repositories
// and provides high-level methods for the application.
type Service struct {
	db        *sql.DB
	logger    *logrus.Logger
	Users     repository.UserRepository
	Todos     repository.TodoRepository
	Comments  repository.CommentRepository
	Families  repository.FamilyRepository
	Calendar  repository.CalendarRepository
	Buying    repository.BuyingListRepository
	WishList  repository.WishListRepository
	Reminders repository.ReminderRepository
}

// New creates a new Service with all required dependencies.
func New(db *sql.DB, logger *logrus.Logger,
	users repository.UserRepository,
	todos repository.TodoRepository,
	comments repository.CommentRepository,
	families repository.FamilyRepository,
	calendar repository.CalendarRepository,
	buying repository.BuyingListRepository,
	wishList repository.WishListRepository,
	reminders repository.ReminderRepository,
) *Service {
	return &Service{
		db: db, logger: logger,
		Users: users, Todos: todos, Comments: comments,
		Families: families, Calendar: calendar, Buying: buying,
		WishList: wishList, Reminders: reminders,
	}
}

// EnsureUser retrieves an existing user by Telegram ID, or creates a new one
// if not found. If the user already exists but their profile information has
// changed (username, first name, last name), it updates the record.
func (s *Service) EnsureUser(ctx context.Context, telegramID int64, username, firstName, lastName string) (*models.User, error) {
	username = strings.TrimSpace(username)
	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)

	user, err := s.Users.GetByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup user (telegram_id=%d): %w", telegramID, err)
	}
	if user == nil {
		// User does not exist yet — create a new record.
		now := time.Now()
		user = &models.User{
			TelegramID:       telegramID,
			TelegramUsername: username,
			FirstName:        firstName,
			LastName:         lastName,
			IsActive:         true,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		user, err = s.Users.Create(ctx, user)
		if err != nil {
			return nil, fmt.Errorf("failed to create user (telegram_id=%d): %w",
				telegramID, err)
		}
		s.logger.Infof("Created new user: %s (telegram_id=%d)", user.DisplayName(), telegramID)
		return user, nil
	}

	// User exists (not nil, no error) — check whether any profile fields need updating.
	needsUpdate := false
	if user.TelegramUsername != username {
		user.TelegramUsername = username
		needsUpdate = true
	}
	if user.FirstName != firstName {
		user.FirstName = firstName
		needsUpdate = true
	}
	if user.LastName != lastName {
		user.LastName = lastName
		needsUpdate = true
	}

	if needsUpdate {
		user.UpdatedAt = time.Now()
		user, err = s.Users.Update(ctx, user)
		if err != nil {
			return nil, fmt.Errorf("failed to update user %d: %w", user.ID, err)
		}
		s.logger.Infof("Updated user profile: %s (telegram_id=%d)", user.DisplayName(), telegramID)
	}

	return user, nil
}

// EnsureFamily retrieves an existing family for the given chat ID, or creates
// a new one if it does not exist. If the chat title has changed, the family
// name is updated accordingly.
func (s *Service) EnsureFamily(ctx context.Context, chatID int64, chatTitle string) (*models.Family, error) {
	chatTitle = strings.TrimSpace(chatTitle)

	family, err := s.Families.GetByChatID(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup family (chat_id=%d): %w", chatID, err)
	}
	if family == nil {
		// Family does not exist yet — create a new record.
		now := time.Now()
		family = &models.Family{
			ChatID:    chatID,
			Name:      chatTitle,
			CreatedAt: now,
			UpdatedAt: now,
		}
		family, err = s.Families.Create(ctx, family)
		if err != nil {
			return nil, fmt.Errorf("failed to create family for chat %d: %w", chatID, err)
		}
		s.logger.Infof("Created new family: %q (chat_id=%d)", chatTitle, chatID)
		return family, nil
	}

	// Family exists — update name if the chat title has changed.
	if chatTitle != "" && family.Name != chatTitle {
		family.Name = chatTitle
		family.UpdatedAt = time.Now()
		family, err = s.Families.Update(ctx, family)
		if err != nil {
			return nil, fmt.Errorf("failed to update family %d: %w", family.ID, err)
		}
		s.logger.Infof("Updated family name to %q (family_id=%d)", chatTitle, family.ID)
	}

	return family, nil
}

// EnsureFamilyMember makes sure the given user is a member of the family
// associated with the specified chat. If the user is already a member, this
// is a no-op. New members are added with the "member" role.
func (s *Service) EnsureFamilyMember(ctx context.Context, familyID int64, userID int64) error {
	members, err := s.Families.GetMembers(ctx, familyID)
	if err != nil {
		return fmt.Errorf("failed to get members for family %d: %w", familyID, err)
	}

	for _, m := range members {
		if m.ID == userID {
			return nil // already a member
		}
	}

	if err := s.Families.AddMember(ctx, familyID, userID, "member"); err != nil {
		return fmt.Errorf("failed to add user %d to family %d: %w", userID, familyID, err)
	}

	s.logger.Infof("Added user %d to family %d", userID, familyID)
	return nil
}
