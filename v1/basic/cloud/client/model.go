package client

import (
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/lazybark/go-pretty-code/logs"
	"github.com/lazycloud-app/go-filesync/v1/basic/fs"
	"github.com/lazycloud-app/go-filesync/v1/basic/proto"
	"github.com/lazycloud-app/go-fsp-proto/ver"
	"gorm.io/gorm"
)

type (
	Client struct {
		AppVersion         ver.AppVersion
		Config             *Config
		Logger             *logs.Logger
		fsEventsChan       chan (fsnotify.Event)
		DB                 *gorm.DB
		wg                 *sync.WaitGroup
		TimeStart          time.Time
		ServerToken        string
		FW                 *fs.Fileprocessor
		CurrentToken       string
		SyncActive         bool
		FileGetter         chan (proto.GetFile)
		EventsChannel      chan (proto.FSEvent)
		FilesInRow         []proto.GetFile
		ActionsBuffer      map[string][]BufferedAction
		ActionsBufferMutex sync.RWMutex
		SessionKey         string
		Stat               *Statistics
		fp                 *fs.Fileprocessor
		syncEventsChan     chan (fs.EventArray) //Channel to notify clients about changes
		syncErrChan        chan (error)
	}

	Config struct {
		Login              string `mapstructure:"LOGIN"`
		Password           string `mapstructure:"PASSWORD"`
		ServerCert         string `mapstructure:"CERT_PATH"`
		Token              string `mapstructure:"TOKEN"`
		ServerAddress      string `mapstructure:"SERVER_ADDRESS"`
		LogDirMain         string `mapstructure:"LOG_DIR_MAIN"`
		CacheDir           string `mapstructure:"CACHE_DIR"`
		ServerPort         int    `mapstructure:"SERVER_PORT"`
		DeviceName         string `mapstructure:"DEVICE_NAME"`
		UserName           string `mapstructure:"USER_NAME"`
		DeviceLabel        string `mapstructure:"DEVICE_LABEL"`
		FileSystemRootPath string `mapstructure:"FILE_SYSTEM_ROOT_PATH"`
		SQLiteDBName       string `mapstructure:"SQLITE_DB_NAME"`
		MaxMessageSize     int
	}

	BufferedAction struct {
		Action    fsnotify.Op
		Skipped   bool
		Timestamp time.Time
	}
)
