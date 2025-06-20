package utils

import "gorm.io/gorm"

// RollbackIfPanic handles rollback if a panic occurred.
// Should be deferred right after tx.Begin()
func RollbackIfPanic(tx *gorm.DB) {
	if err := recover(); err != nil {
		tx.Rollback()
		panic(err)
	}
}
