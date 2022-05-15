package server

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/lazybark/go-pretty-code/logs"
	"github.com/lazycloud-app/go-filesync/helpers"
	u "github.com/lazycloud-app/go-filesync/users"
	"github.com/lazycloud-app/go-filesync/v1/basic/fs"
	"github.com/lazycloud-app/go-filesync/v1/basic/proto"
	"go.uber.org/zap"
)

func NewServer() (*Server, error) {
	//The time server was created (we suppose it will be called imidiately)
	timeInit := time.Now()
	//Get config
	var conf Config
	err := helpers.LoadConfig(".", &conf)
	if err != nil {
		return nil, fmt.Errorf("[NewServer] LoadConfig failed: %w", err)
	}
	//Connect logger
	logfileName := fmt.Sprintf("go-filesync_%v-%v-%v_%v-%v-%v.log", timeInit.Year(), timeInit.Month(), timeInit.Day(), timeInit.Hour(), timeInit.Minute(), timeInit.Second())
	logger, err := logs.Double(filepath.Join(conf.LogDirMain, logfileName), false, zap.InfoLevel)
	if err != nil {
		log.Fatal("[NewServer] unable to make logger: ", err)
	}
	//Connect DB
	db, err := helpers.OpenSQLite(conf.SQLiteDBName)
	if err != nil {
		return nil, fmt.Errorf("[NewServer] OpenSQLite failed: %w", err)
	}
	//Start FS watcher
	watch, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("[NewServer] NewWatcher failed: %w", err)
	}
	//Deploy new cloud server
	var cpm sync.RWMutex
	var spm sync.RWMutex
	sdata := ServerData{proto.ServerVersion, proto.ServerVersionLabel, proto.ALServerBase, conf.HostName, conf.OwnerContacts, conf.FileSystemRootPath}
	//New fileprocessor to work with FS changes provided by watcher
	fp := fs.NewProcessor(conf.FileSystemRootPath, watch, db)

	s := &Server{
		timeInit,
		sdata,
		&conf,
		db,
		make(chan bool),
		make(chan fs.EventArray),
		make(chan error),
		&cpm,
		new(sync.WaitGroup),
		fp,
		&CommPool{make(map[string]*Comm), &spm, sdata, db, fp, *logger},

		*logger,
	}

	return s, nil
}

func (s *Server) Start() {
	s.InfoCyan(fmt.Sprintf("App Version: %s", s.sData.appVersion))
	// Clear old file records in case real files have been changed since last scanning
	s.InitDB()
	// Watch for filesystem events
	go s.NotificationPublisher()
	go s.fp.FilesystemWatcherRoutine(s.syncEventsChan, s.syncErrChan)
	// Process and watch all subdirs
	s.processRoot()
	// Controls server instance
	go s.AdminRoutine()
	// Counts current stats and stores in db
	go s.CountStats()
	// Listens for new connections from clients
	go s.Listen()
}

func (s *Server) processRoot() {
	s.InfoGreen("Processing root directory")
	rpStart := time.Now()
	files, dirs, err, errs := s.fp.ProcessDirectory(s.config.FileSystemRootPath)
	if err != nil {
		s.Error(fmt.Errorf("error processing directory: %w", err))
	}
	if len(errs) > 0 {
		text := ""
		for _, e := range errs {
			text += e.Error() + "\n"
		}
		s.Warn("Errors in processing filesystem: \n" + text)
	}
	s.InfoGreen(fmt.Sprintf("Root directory processed. Total %d files in %d directories. Time: %v", files, dirs, time.Since(rpStart)))
	s.InfoGreen("Starting server")
}

func (s *Server) closeConnections() {
	for _, c := range s.pool.pool {
		if !c.active { //
			continue
		}
		c.active = false
		c.closedByServer = true
		c.stateChan <- ConnClose
		s.Info(fmt.Sprintf("(%v) closed", c.IP()))
	}
}

func (s *Server) Stop() {
	s.serverDoneChan <- true
}

func (s *Server) AdminRoutine() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	var sig os.Signal
	for {
		select {
		case sig = <-c:
			s.InfoRed(fmt.Sprintf("Got %s signal", sig))
			s.InfoRed("Stopping server")
			// Close filesystem watcher
			err := s.fp.Watcher.Close()
			if err != nil {
				s.Error(fmt.Errorf("[AdminRoutine] error closing fs watcher: %w", err))
			}
			// Signal active clients that server will be stopped
			s.closeConnections()
			// Log server closing
			s.InfoBackRed(fmt.Sprintf("Server stopped. Was online for %v", time.Since(s.timeStart)))
			//s.Close()
			os.Exit(1)
		case done := <-s.serverDoneChan:
			if !done {
				continue
			}
			// Send active clients connection breaker
			s.InfoRed("Stopping server")
			// TO DO: add Listen breaker
			// here
			// Close filesystem watcher
			s.fp.Watcher.Close()
			// Signal clients that server will be stopped
			s.closeConnections()
			// Log server closing
			s.InfoBackRed(fmt.Sprintf("Server stopped. Was online for %v", time.Since(s.timeStart)))
			//s.Close()

		}
	}

}

func (s *Server) InitDB() (err error) {
	err = s.db.Migrator().DropTable(&fs.File{}, &fs.Folder{})
	if err != nil {
		return err
	}
	err = s.db.AutoMigrate(&u.User{}, &u.Client{}, &fs.File{}, &fs.Folder{}, &Statistics{}, &StatisticsBySession{})

	return
}
