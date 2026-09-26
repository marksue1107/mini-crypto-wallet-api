package entity

import "gorm.io/gorm"

type DBClient struct {
	MasterDB *gorm.DB
}
