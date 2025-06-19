package repositories

import "mini-crypto-wallet-api/models"

type UserRepository interface {
	CreateUser(user *models.User) error
}
