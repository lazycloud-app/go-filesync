package proto

import "time"

type (
	Filesystem struct {
		Folders []Folder
		Files   []File
	}

	// File represents file data into DB
	File struct {
		ID            uint `gorm:"primaryKey"`
		Hash          string
		Name          string `gorm:"uniqueIndex:file"`
		Path          string `gorm:"uniqueIndex:file"`
		Owner         uint
		Size          int64
		FSUpdatedAt   time.Time
		CreatedAt     time.Time
		UpdatedAt     time.Time
		CurrentStatus string
		LocationDirId int
		Type          string
	}

	// Folder represents folder data to exchange current sync status information
	Folder struct {
		ID            uint   `gorm:"primaryKey"`
		Name          string `gorm:"uniqueIndex:folder"`
		Path          string `gorm:"uniqueIndex:folder"`
		Owner         uint
		FSUpdatedAt   time.Time
		CreatedAt     time.Time
		UpdatedAt     time.Time
		CurrentStatus string
		LocationDirId int
		Items         int
		Size          int64
	}
)
