package dao

import (
	"context"
	"gorm.io/gorm"
)

type Mark struct {
	Inited bool `gorm:"type:tinyint(1)"`
}

func InitTables(db *gorm.DB) {
	db.AutoMigrate(&Mark{})
	db.AutoMigrate(&ShortUrl{})
	db.WithContext(context.Background()).Create(&Mark{
		Inited: true,
	})
}
