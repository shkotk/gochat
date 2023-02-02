package repositories

import (
	"context"
	"errors"

	"github.com/shkotk/gochat/server/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type UserRepository struct {
	logger *logrus.Logger
	db     *gorm.DB
}

func NewUserRepository(logger *logrus.Logger, db *gorm.DB) *UserRepository {
	return &UserRepository{logger, db}
}

func (r *UserRepository) Exists(ctx context.Context, username string) (exists bool, err error) {
	err = r.db.WithContext(ctx).
		Model(&models.User{}).
		Select("count(*) > 0").
		Where("username = ?", username).
		Find(&exists).
		Error
	if err != nil {
		r.logger.WithError(err).
			WithFields(logrus.Fields{
				"action":    "check_if_user_exists",
				"record_id": username,
			}).
			Error()
	}

	return
}

func (r *UserRepository) Create(ctx context.Context, user models.User) error {
	err := r.db.WithContext(ctx).Create(&user).Error
	if err != nil {
		r.logger.WithError(err).
			WithFields(logrus.Fields{
				"action":    "create_user",
				"record_id": user.Username,
			}).
			Error()
		return err
	}

	return nil
}

// Returns user corresponding to provided username or nil if it does not exist.
func (r *UserRepository) Get(ctx context.Context, username string) (*models.User, error) {
	user := &models.User{}
	err := r.db.WithContext(ctx).First(user, "username = ?", username).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		r.logger.WithError(err).
			WithFields(logrus.Fields{
				"action":    "get_user",
				"record_id": username,
			}).
			Error()
		return nil, err
	}

	return user, nil
}
