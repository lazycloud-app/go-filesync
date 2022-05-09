package proto

import (
	"time"
)

type (

	// REDUNDANT
	DirSyncReq struct {
		Token      string
		Root       string // Directory path that needs to be synced
		Filesystem Filesystem
	}

	// REDUNDANT
	DirSyncResp struct {
		Token          string
		Filesystem     Filesystem
		UploadToServer []string
	}

	SyncFileData struct {
		Hash        string
		Name        string
		Path        string
		Size        int64
		FSUpdatedAt time.Time
		Type        string
		Data        []byte
	}

	SyncDirData struct {
		Id            int
		Name          string
		Path          string
		CurrentStatus string
		LocationDirId int
		Data          []byte
	}
)
