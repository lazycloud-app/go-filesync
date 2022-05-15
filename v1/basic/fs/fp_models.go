package fs

import (
	"fmt"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/lazycloud-app/go-filesync/v1/basic/proto"
	"gorm.io/gorm"
)

type (
	FSObjectType int
	FSEventType  int

	ErrType int

	BufferedAction struct {
		Action    FSEventType
		Ignore    bool
		Processed bool
		Timestamp time.Time
	}

	EventArray struct {
		Proto proto.FSEvent
		FS    FSObject
	}

	ProcessingError struct {
		Type    ErrType
		ErrList []error
	}

	FSEvent struct {
		Type FSEventType
		At   time.Time
	}

	Fileprocessor struct {
		root    string
		Watcher *fsnotify.Watcher
		db      *gorm.DB
		//actionBuffer holds actions that need to be considered during processing other actions.
		//Key is the object full address
		actionBuffer map[string][]BufferedAction
		abMutex      *sync.RWMutex
	}

	Filesystem struct {
		Folders []Folder
		Files   []File
	}

	// File represents file data into DB
	File struct {
		ID          uint `gorm:"primaryKey"`
		Hash        string
		Name        string `gorm:"uniqueIndex:file"`
		Path        string `gorm:"uniqueIndex:file"`
		Owner       uint
		Size        int64
		Ext         string
		FSUpdatedAt time.Time
		CreatedAt   time.Time
		UpdatedAt   time.Time
		IsRenamed   bool
	}

	// Folder represents folder data to exchange current sync status information
	Folder struct {
		ID          uint `gorm:"primaryKey"`
		Hash        string
		Name        string `gorm:"uniqueIndex:folder"`
		Path        string `gorm:"uniqueIndex:folder"`
		Owner       uint
		Size        int64
		Items       int
		FSUpdatedAt time.Time
		CreatedAt   time.Time
		UpdatedAt   time.Time
		IsRenamed   bool
	}

	FSObject struct {
		Type        FSObjectType
		ID          uint `gorm:"primaryKey"`
		Hash        string
		Name        string `gorm:"uniqueIndex:file"`
		Path        string `gorm:"uniqueIndex:file"`
		Owner       uint
		Size        int64
		Items       int
		Ext         string
		FSUpdatedAt time.Time
		CreatedAt   time.Time
		UpdatedAt   time.Time
		Events      []FSEvent
	}
)

const (
	FS_NO_ACTION FSEventType = iota
	FS_CREATED
	FS_DELETED
	FS_UPDATED
	FS_RENAMED
	FS_UNKNOWN_ACTION
	FS_ANY_ACTION
)

const (
	ERR_UNKNOWN ErrType = iota
	ERR_DIRECTORY_PROCESSING
	ERR_OBJECT_NOT_TRACKED
	ERR_NEWER_VERSION_EXISTS
)

func (e ProcessingError) Error() string {
	return fmt.Sprintf("processed with %d errors", len(e.ErrList))
}

func (e ErrType) String() string {
	return [...]string{"Unknown error", "Directory processing error", "The object was never tracked"}[e]
}

func (t FSEventType) String() string {
	return [...]string{"NO_ACTION", "CREATED", "DELETED", "UPDATED", "RENAMED", "UNKNOWN", "ANY"}[t]
}

//Proto() returns event type according to proto version
//Protocol implementiong apps may have more or even less possible events then protocol describes,
//so they need converter
func (t FSEventType) Proto() proto.FSEventType {
	return [...]proto.FSEventType{proto.FS_UNKNOWN_ACTION, proto.FS_CREATED, proto.FS_DELETED, proto.FS_UPDATED, proto.FS_RENAMED}[t]
}

const (
	OBJ_UNKNOWN FSObjectType = iota
	OBJ_FILE
	OBJ_DIR
)

func (t FSObjectType) String() string {
	return [...]string{"Unknown filesystem objet type", "FILE", "DIR"}[t]
}

func (t FSObjectType) Proto() proto.FSObjectType {
	return [...]proto.FSObjectType{proto.OBJ_UNKNOWN, proto.OBJ_FILE, proto.OBJ_DIR}[t]
}
