package md

import "time"

type (
	//File represents file data into DB
	File struct {
		ID          uint `gorm:"primaryKey"`
		Hash        string
		Name        string `gorm:"uniqueIndex:file"`
		Path        string `gorm:"uniqueIndex:file"`
		Owner       uint
		Size        int64
		Ext         string
		FSUpdatedAt time.Time
		CreatedAt   time.Time
		UpdatedAt   time.Time
		IsRenamed   bool
	}

	//Folder represents folder data into DB
	Folder struct {
		ID          uint `gorm:"primaryKey"`
		Hash        string
		Name        string `gorm:"uniqueIndex:folder"`
		Path        string `gorm:"uniqueIndex:folder"`
		Owner       uint
		Size        int64
		Items       int
		FSUpdatedAt time.Time
		CreatedAt   time.Time
		UpdatedAt   time.Time
		IsRenamed   bool
	}
)
