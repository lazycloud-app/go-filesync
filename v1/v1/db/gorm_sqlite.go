package db

import (
	"log"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gLogger "gorm.io/gorm/logger"
)

func NewGormSQLite(dbName string) (*GORM, error) {
	g := new(GORM)

	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{Logger: gLogger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		gLogger.Config{LogLevel: gLogger.Silent},
	)})

	g.SetDB(db)

	return g, err
}
