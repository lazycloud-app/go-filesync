package server

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/lazycloud-app/go-filesync/v1/basic/cloud/events"
)

type ()

// NewServer creates a sync server instance

func (s *Server) Start() {
	/*s.timeStart = time.Now()
	//s.appVersion = proto.ServerVersion
	//s.appVersionLabel = proto.ServerVersionLabel
	s.appLevel = ALServerBase
	s.ewg = &sync.WaitGroup{}
	s.externalStartedChan = started
	s.serverDoneChan = make(chan bool)
	s.activeSessions = make(map[uint][]*Session)*/
	// Get config into config.Current struct

	// Catch events
	//s.evProc = events.NewStandartLogsProcessor(filepath.Join(s.config.LogDirMain, s.LogfileName()), true)
	// Log current version
	s.Send(EventType("cyan"), events.SourceSyncServer.String(), fmt.Sprintf("App Version: %s", s.appVersion))
	// Launch stat logger routine
	if s.config.LogStats {
		go s.LogStats()
	}
	/*// Connect DB
	s.db, err = cloud.OpenSQLite(s.config.SQLiteDBName)
	if err != nil {
		s.evProc.Send(EventType("fatal"), events.SourceSyncServer.String(), fmt.Errorf("[Start] OpenSQLite failed: %w", err))
	}*/
	// Start FSEventsProcessor + connection pool manager
	//s.fsEventsChan = make(chan fsnotify.Event)
	go s.FSEventsProcessor()
	// Connect FS watcher
	/*watch, err := fsnotify.NewWatcher()
	if err != nil {
		s.evProc.Send(EventType("fatal"), events.SourceSyncServer.String(), fmt.Errorf("new Watcher failed: %w", err))
	}
	s.watcher = watch*/
	// Force rescan filesystem and flush old DB-records
	s.InitDB()
	/*if err != nil {
		s.evProc.Send(EventType("fatal"), events.SourceSyncServer.String(), fmt.Errorf("error in InitDB: %w", err))
	}*/
	// New filesystem worker
	//s.fw = fsworker.NewWorker(s.config.FileSystemRootPath, s.db, watch)

	//users.CreateUserCLI(s.DB)

	// Watch root dir
	go s.FilesystemWatcherRoutine()
	// Process and watch all subdirs
	s.SendVerbose(EventType("green"), events.SourceSyncServer.String(), "Processing root directory")
	rpStart := time.Now()
	files, dirs, err, errs := s.fw.ProcessDirectory(s.config.FileSystemRootPath)
	if err != nil {
		s.Send(EventType("error"), events.SourceSyncServer.String(), fmt.Errorf("error processing directory: %w", err))
	}
	if len(errs) > 0 {
		text := ""
		for _, e := range errs {
			text += e.Error() + "\n"
		}
		s.Send(EventType("warn"), events.SourceSyncServer.String(), "Errors in processing filesystem: \n"+text)
	}
	s.SendVerbose(EventType("green"), events.SourceSyncServer.String(), fmt.Sprintf("Root directory processed. Total %d files in %d directories. Time: %v", files, dirs, time.Since(rpStart)))
	s.Send(EventType("green"), events.SourceSyncServer.String(), "Starting server")

	// Controls server instance
	go s.AdminRoutine()
	// Counts current stats and stores in db
	go s.CountStats()
	// Listens for new connections from clients
	go s.Listen()
}

func (s *Server) LogStats() {
	for {
		time.Sleep(5 * time.Minute)
		s.Send(EventType("magenta"), events.SourceSyncServerStats.String(), "Server stats: \n - active users = 0\n - active connections = 0\n - data recieved = 0 Gb\n - data sent = 0 Gb\n - errors last 15 min / hour / 24 hours = 0/0/0")
	}
}

func (s *Server) closeConnections() {
	for _, c := range s.activeConnections {
		if !c.Active { //
			continue
		}
		c.Active = false
		c.ClosedByServer = true
		c.StateChan <- ConnClose
		s.SendVerbose(EventType("info"), events.SourceAdminRoutine.String(), fmt.Sprintf("(%v) closed", c.ip))
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
			s.Send(EventType("red"), events.SourceAdminRoutine.String(), fmt.Sprintf("Got %s signal", sig))
			s.SendVerbose(EventType("red"), events.SourceSyncServer.String(), "Stopping server")
			// Close filesystem watcher
			err := s.watcher.Close()
			if err != nil {
				s.Send(EventType("error"), events.SourceAdminRoutine.String(), fmt.Errorf("[AdminRoutine] error closing fs watcher: %w", err))
			}
			// Signal active clients that server will be stopped
			s.closeConnections()
			// Log server closing
			s.Send(EventType("backred"), events.SourceAdminRoutine.String(), fmt.Sprintf("Server stopped. Was online for %v", time.Since(s.timeStart)))
			s.Close()
			os.Exit(1)
		case done := <-s.serverDoneChan:
			if !done {
				continue
			}
			// Send active clients connection breaker
			s.SendVerbose(EventType("red"), events.SourceAdminRoutine.String(), "Stopping server")
			// TO DO: add Listen breaker
			// here
			// Close filesystem watcher
			s.watcher.Close()
			// Signal clients that server will be stopped
			s.closeConnections()
			// Log server closing
			s.Send(EventType("backred"), events.SourceAdminRoutine.String(), fmt.Sprintf("Server stopped. Was online for %v", time.Since(s.timeStart)))
			s.Close()

		}
	}

}

func (s *Server) LogfileName() string {
	return fmt.Sprintf("go-filesync_%v-%v-%v_%v-%v-%v.log", s.timeStart.Year(), s.timeStart.Month(), s.timeStart.Day(), s.timeStart.Hour(), s.timeStart.Minute(), s.timeStart.Second())
}
