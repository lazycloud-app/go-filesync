package proto

import (
	"time"
)

type (
	SyncFileData struct {
		Hash        string
		Name        string
		Path        string
		Size        int64
		FSUpdatedAt time.Time
		Type        string
	}

	SyncDirData struct {
		Id   int
		Name string
		Path string
	}
)
