package proto

import "time"

type (
	FSEventType  int
	FSObjectType int

	FSEvent struct {
		Action FSEventType
		Object FSObject
	}

	FSObject struct {
		ObjectType FSObjectType
		FullPath   string
		Hash       string
		Ext        string
		Size       int64
		UpdatedAt  time.Time
	}
)

const (
	//NO_ACTION represents an empty value. For example, when there is no need for other party to know about event
	FS_NO_ACTION FSEventType = iota
	//FS_CREATED means that some object was created in the filesystem
	FS_CREATED
	//FS_DELETED means that some object was deleted from the filesystem.
	//v1 of the protocol does not define deletion type ('to trash', etc.)
	FS_DELETED
	//FS_UPDATED represents any possible update to any object in the filesystem.
	//v1 of the protocol has no clarifying statements about exact update parameters
	FS_UPDATED
	//FS_RENAMED means that some object was renamed
	//(and also can be processed like deleted->created series of actions by some clients)
	FS_RENAMED
	//FS_UNKNOWN_ACTION represents action that is beyond this protocol version
	FS_UNKNOWN_ACTION
	//FS_ANY_ACTION represents state when any action needs same response or processing
	FS_ANY_ACTION
)

func (t FSEventType) String() string {
	return [...]string{"NO_ACTION", "CREATED", "DELETED", "UPDATED", "RENAMED", "UNKNOWN", "ANY"}[t]
}

const (
	OBJ_UNKNOWN FSObjectType = iota
	OBJ_FILE
	OBJ_DIR
)

func (t FSObjectType) String() string {
	return [...]string{"Unknown filesystem objet type", "FILE", "DIR"}[t]
}
