package db

import (
	"github.com/lazycloud-app/go-filesync/v1/v1/md"
	"gorm.io/gorm"
)

type (
	GORM struct {
		db *gorm.DB
	}
)

func (g *GORM) SetDB(d *gorm.DB) {
	g.db = d
}

func (g *GORM) InitClient() error {
	//We drop tables in case there are any unpredictable changes
	err := g.db.Migrator().DropTable(&md.File{}, &md.Folder{})
	if err != nil {
		return err
	}
	//And then create tables again
	err = g.db.AutoMigrate(&md.File{}, &md.Folder{})
	if err != nil {
		return err
	}
	return nil
}

func (g *GORM) RecordDir(record []md.Folder) error {
	if err := g.db.Model(&md.Folder{}).Save(&record).Error; err != nil && err != gorm.ErrEmptySlice {
		return err
	}
	return nil
}

func (g *GORM) RecordFile(record []md.File) error {
	if err := g.db.Model(&md.File{}).Save(&record).Error; err != nil && err != gorm.ErrEmptySlice {
		return err
	}
	return nil
}
