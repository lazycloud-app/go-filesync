package db

import "github.com/lazycloud-app/go-filesync/v1/v1/md"

type (
	//DataBase is the interface to process filesystem changes in app database.
	//It abstracts ORM or DB package to simple methods, so driver change will not affect whole app.
	//
	//So app may work with any database or even without it (if you dare)
	DataBase interface {
		//Init initializes database for app
		InitClient() error
		//RecordDir saves dir data into DB
		RecordDir(record []md.Folder) error
		//RecordFile saves file data into DB
		RecordFile(record []md.File) error
	}
)
