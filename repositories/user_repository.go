package repositories

import (
	"mini-crypto-wallet-api/db_conn"
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/repositories/entity"
)

type userRepository struct {
	entity.DBClient
}

func NewUserRepository() IUser {
	r := new(userRepository)
	r.DBClient.MasterDB = db_conn.Conn_DB.MasterDB

	return r
}

func (r *userRepository) CreateUser(user *models.User) error {
	return r.DBClient.MasterDB.Create(user).Error
}
