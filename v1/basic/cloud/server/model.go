package server

import (
	"sync"
	"time"

	"github.com/lazybark/go-pretty-code/logs"
	"github.com/lazycloud-app/go-filesync/v1/basic/fs"
	"github.com/lazycloud-app/go-filesync/v1/basic/proto"
	"github.com/lazycloud-app/go-fsp-proto/ver"
	"gorm.io/gorm"
)

type (
	Server struct {
		timeStart           time.Time
		sData               ServerData
		config              *Config
		db                  *gorm.DB
		serverDoneChan      chan (bool)          // serverDone signals that server has been stopped
		syncEventsChan      chan (fs.EventArray) //Channel to notify clients about changes
		syncErrChan         chan (error)
		connectionPoolMutex *sync.RWMutex   // Mutex for writing into activeConnections
		ewg                 *sync.WaitGroup // waitgroup for events processor, in case server interrupt will happen before all events are done
		fp                  *fs.Fileprocessor
		pool                *CommPool

		logs.Logger
	}

	ServerData struct {
		appVersion      ver.AppVersion
		appVersionLabel string
		appLevel        proto.AppLevel
		name            string
		ownerContacts   string
		rootDir         string
	}

	Config struct {
		ServerToken              string `mapstructure:"SERVER_TOKEN"`
		CertPath                 string `mapstructure:"CERT_PATH"`
		KeyPath                  string `mapstructure:"KEY_PATH"`
		HostName                 string `mapstructure:"HOST_NAME"`
		Port                     string `mapstructure:"PORT"`
		MaxClientErrors          uint   // Limit for client-side errors (or any other party) until problematic connection will be closed and ErrTooMuchClientErrors sent
		MaxServerErrors          uint   // Limit for server-side errors until problematic connection will be closed and ErrTooMuchServerErrors sent
		LogStats                 bool
		CollectStats             bool
		TokenValidDays           int
		ServerVerboseLogging     bool   `mapstructure:"SERVER_VERBOSE_LOGGING"`
		CountStats               bool   `mapstructure:"COUNT_STATS"`
		FilesystemVerboseLogging bool   `mapstructure:"FILESYSTEM_VERBOSE_LOGGING"`
		SilentMode               bool   `mapstructure:"SILENT_MODE"`
		LogDirMain               string `mapstructure:"LOG_DIR_MAIN"`
		FileSystemRootPath       string `mapstructure:"FILE_SYSTEM_ROOT_PATH"`
		SQLiteDBName             string `mapstructure:"SQLITE_DB_NAME"`
		ServerName               string `mapstructure:"SERVER_NAME"`
		OwnerContacts            string `mapstructure:"OWNER_CONACTS"`
		MaxConnectionsPerUser    int    `mapstructure:"MAX_USER_CONNECTIONS_PER_USER"`
	}
)
