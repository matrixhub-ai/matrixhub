package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/crypto"
)

type UserRepository struct {
	db *gorm.DB
}

func (u *UserRepository) CreateUser(ctx context.Context, user user.User) error {
	password, err := crypto.HashPassword(user.Password)
	if err != nil {
		return err
	}
	user.Password = password
	return u.db.WithContext(ctx).Create(&user).Error
}

func (u *UserRepository) GetUser(ctx context.Context, id string) (*user.User, error) {
	var user user.User
	err := u.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (u *UserRepository) ListUsers(ctx context.Context, page, pageSize int, search string) (us []*user.User, total int64, err error) {
	query := u.db.WithContext(ctx).Limit(pageSize).Offset((page - 1) * pageSize)
	if search != "" {
		query = query.Where("name LIKE ?", "%"+search+"%")
	}
	if err = query.Count(&total).Error; err != nil {
		return
	}
	err = query.Find(&us).Error
	return
}

func (u *UserRepository) DeleteUser(ctx context.Context, id string) error {
	return u.db.WithContext(ctx).Where("id = ?", id).Delete(&user.User{}).Error
}

func NewUserRepository(db *gorm.DB) user.IUserRepository {
	return &UserRepository{
		db: db,
	}
}
