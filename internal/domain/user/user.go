package user

import (
	"context"
	"time"
)

type User struct {
	ID        string `gorm:"primarykey"`
	Username  string
	Password  string
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (User) TableName() string {
	return "users"
}

type IUserRepository interface {
	CreateUser(ctx context.Context, user User) error
	GetUser(ctx context.Context, id string) (*User, error)
	ListUsers(ctx context.Context, page, pageSize int, search string) ([]*User, int64, error)
	DeleteUser(ctx context.Context, id string) error
}
