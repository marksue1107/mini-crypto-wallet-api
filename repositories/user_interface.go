package repositories

import "mini-crypto-wallet-api/models"

type IUser interface {
	CreateUser(user *models.User) error
	GetUserByUsername(username string) (*models.User, error)
	GetUserByID(userID uint) (*models.User, error)
}
