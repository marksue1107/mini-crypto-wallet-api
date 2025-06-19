package repositories

import (
	"mini-crypto-wallet-api/database"
	"mini-crypto-wallet-api/models"
)

type userRepositoryImpl struct{}

func NewUserRepository() UserRepository {
	return &userRepositoryImpl{}
}

func (r *userRepositoryImpl) CreateUser(user *models.User) error {
	return database.DB.Create(user).Error
}
