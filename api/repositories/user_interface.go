package repositories

import (
	"mini-crypto-wallet-api/models"

	"gorm.io/gorm"
)

type IUser interface {
	CreateUser(user *models.User, tx ...*gorm.DB) error
	GetUserByUsername(username string) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	GetUserByID(userID uint) (*models.User, error)
}
