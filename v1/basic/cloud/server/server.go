package server

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/lazycloud-app/go-filesync/v1/basic/cloud"
	"github.com/lazycloud-app/go-filesync/v1/basic/cloud/events"
	"github.com/lazycloud-app/go-filesync/v1/basic/fsworker"
	"github.com/lazycloud-app/go-filesync/v1/basic/proto"
)

func NewServer() *Server {
	timeInit := time.Now()

	conf := Config{}
	err := LoadConfig(".", &conf)
	if err != nil {
		log.Fatal("Error getting config: ", err)
	}

	LogfileName := fmt.Sprintf("go-filesync_%v-%v-%v_%v-%v-%v.log", timeInit.Year(), timeInit.Month(), timeInit.Day(), timeInit.Hour(), timeInit.Minute(), timeInit.Second())

	// Connect DB
	db, err := cloud.OpenSQLite(conf.SQLiteDBName)
	if err != nil {
		log.Fatal("Error getting db: ", err)
		//	s.evProc.Send(EventType("fatal"), events.SourceSyncServer.String(), fmt.Errorf("[Start] OpenSQLite failed: %w", err))
	}

	watch, _ := fsnotify.NewWatcher()

	var cpm sync.RWMutex
	var spm sync.RWMutex
	var cpool []*Connection

	evp := events.NewStandartLogsProcessor(filepath.Join(conf.LogDirMain, LogfileName), true)

	s := &Server{
		proto.ServerVersion,
		proto.ServerVersionLabel,
		proto.ALServerBase,
		&conf,
		db,
		make(chan fsnotify.Event),
		make(chan bool),
		&cpm,
		cpool,
		0,
		&spm,
		make(map[uint][]*Session),
		0,
		watch,
		timeInit,
		fsworker.NewWorker(conf.FileSystemRootPath, db, watch),
		new(sync.WaitGroup),

		evp,
	}

	return s
}
