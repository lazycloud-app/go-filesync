package proto

import (
	"time"
)

type (
	SyncEventType int

	SyncObject int

	AppLevel int

	// SyncType represents sync types that app instance can be used for
	SyncType int

	SyncEvent struct {
		Type         SyncEventType // What happened: created, deleted, updated
		ObjectType   SyncObject    // What object involved: dir or file
		Name         string
		Path         string
		Hash         string
		NewUpdatedAt time.Time
	}

	GetFile struct {
		Name      string
		Path      string
		Hash      string
		UpdatedAt time.Time
	}
)

func (e SyncEventType) String() string {
	if e <= sync_events_start || sync_events_end <= e {
		return "Illegal SyncEvent"
	}
	return [...]string{"Created", "Updated", "Deleted"}[e-1]
}

const (
	sync_objects_start SyncObject = iota

	ObjectDir
	ObjectFile

	sync_objects_end
)

func (o SyncObject) String() string {
	if o <= sync_objects_start || sync_objects_end <= o {
		return "Illegal SyncObject"
	}
	return [...]string{"Directory", "File"}[o-1]
}

func (e *SyncEvent) CheckObject() bool {
	if sync_objects_start < e.ObjectType && e.ObjectType < sync_objects_end {
		return true
	}
	return false
}

const (
	sync_events_start SyncEventType = iota

	ObjectCreated
	ObjectUpdated
	ObjectRemoved
	ObjectRenamed

	sync_events_end
	SyncEventIllegal // Just for readability
)

func (ep *SyncEvent) CheckType() bool {
	if sync_events_start < ep.Type && ep.Type < sync_events_end {
		return true
	}
	return false
}

const (
	sync_types_start SyncType = iota

	SyncTypeFullSignal
	SyncTypeFullRequest
	SyncTypeDirectorySignal
	SyncTypeDirectoryRequest
	SyncTypeFileSignal
	SyncTypeFileRequest
	SyncTypeActionSignal
	SyncTypeActionRequest

	sync_types_end
	// Just for readability
	SyncTypeUnknown
)

const (
	ALServerBase AppLevel = iota + 1
	ALServerBackup
)

func (l AppLevel) String() string {

	levelNames := [...]string{"Unknown", "Base sync server", "Backup server"}
	if l > AppLevel(len(levelNames)) {
		return "Unknown"
	}

	return levelNames[l]
}
