package server

import (
	"net"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/lazycloud-app/go-filesync/v1/basic/cloud/events"
	"github.com/lazycloud-app/go-filesync/v1/basic/fsworker"
	"github.com/lazycloud-app/go-filesync/v1/basic/proto"
	"github.com/lazycloud-app/go-filesync/ver"
	"gorm.io/gorm"
)

type (
	Server struct {
		appVersion              ver.AppVersion
		appVersionLabel         string
		appLevel                proto.AppLevel
		config                  *Config
		db                      *gorm.DB
		fsEventsChan            chan (fsnotify.Event) // Channel to send events that occur in the watcher filesystem
		serverDoneChan          chan (bool)           // serverDone signals that server has been stopped
		connectionPoolMutex     *sync.RWMutex         // Mutex for writing into activeConnections
		activeConnections       []*Connection
		activeConnectionsNumber int
		sessionsPoolMutex       *sync.RWMutex // Mutex for writing into activeConnections
		activeSessions          map[uint][]*Session
		activeSessionsNumber    int
		watcher                 *fsnotify.Watcher
		timeStart               time.Time
		fw                      *fsworker.Fsworker
		ewg                     *sync.WaitGroup // waitgroup for events processor, in case server interrupt will happen before all events are done

		events.EventProcessor
	}

	Session struct {
		Uid        uint
		Key        string
		RestrictIP bool
		IP         net.Addr
	}
)
