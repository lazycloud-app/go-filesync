package imp

import (
	"encoding/json"
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/lazycloud-app/go-filesync/v1/basic/proto"
)

func SyncEventFromWatcherEvent(watcherEvent fsnotify.Op) proto.SyncEventType {
	if watcherEvent == fsnotify.Create {
		return proto.ObjectCreated
	} else if watcherEvent == fsnotify.Remove || watcherEvent == fsnotify.Rename {
		return proto.ObjectRemoved
	} else if watcherEvent == fsnotify.Write {
		return proto.ObjectUpdated
	}

	// Return this kind just for better code readability
	return proto.SyncEventIllegal
}

func ParseSyncEvent(raw []byte) (p proto.FSEvent, err error) {
	err = json.Unmarshal(raw, &p)
	if err != nil {
		return p, fmt.Errorf("[ParseSyncEvent] error unmarshalling -> %w", err)
	}
	fmt.Println(p)
	if p.Object.FullPath == "" {
		err = fmt.Errorf("no object found")
		return
	}

	/*if !p.CheckType() {
		err = fmt.Errorf("unknown event type")
		return
	}

	if !p.CheckObject() {
		err = fmt.Errorf("unknown object type")
		return
	}*/
	return
}
