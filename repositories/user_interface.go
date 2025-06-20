package repositories

import "mini-crypto-wallet-api/models"

type IUser interface {
	CreateUser(user *models.User) error
}
