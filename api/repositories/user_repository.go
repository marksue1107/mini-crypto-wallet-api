package repositories

import (
	"mini-crypto-wallet-api/db_conn"
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/repositories/entity"

	"gorm.io/gorm"
)

type userRepository struct {
	entity.DBClient
}

func NewUserRepository() IUser {
	r := new(userRepository)
	r.DBClient.MasterDB = db_conn.Conn_DB.MasterDB

	return r
}

func (r *userRepository) CreateUser(user *models.User, tx ...*gorm.DB) error {
	var db *gorm.DB = r.DBClient.MasterDB
	if len(tx) > 0 {
		db = tx[0]
	}

	return db.Create(user).Error
}

func (r *userRepository) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	if err := r.DBClient.MasterDB.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	if err := r.DBClient.MasterDB.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetUserByID(userID uint) (*models.User, error) {
	var user models.User
	if err := r.DBClient.MasterDB.Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
