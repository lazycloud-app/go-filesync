package fs

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/lazycloud-app/go-filesync/v1/basic/proto"
)

func (p *Fileprocessor) ConvertFromProtoObject(o proto.FSObjectType) FSObjectType {
	if o == proto.OBJ_DIR {
		return OBJ_DIR
	} else if o == proto.OBJ_FILE {
		return OBJ_FILE
	}
	return OBJ_UNKNOWN
}

//ConvertFromProtoEvent returns filesystem object description filled with additional DB data
func (p *Fileprocessor) ConvertFromProtoEvent(e proto.FSEvent) (FSObject, error) {
	var o FSObject
	var folder Folder
	var file File

	o.Type = p.ConvertFromProtoObject(e.Object.ObjectType)
	d, c := p.SplitPath(e.Object.FullPath)

	if e.Action == proto.FS_CREATED {
		o.FSUpdatedAt = e.Object.UpdatedAt
		o.Ext = e.Object.Ext
		o.Hash = e.Object.Hash
		o.Size = e.Object.Size
		o.Path = d
		o.Name = c

		return o, nil
	}

	if e.Object.ObjectType == proto.OBJ_DIR {

		err := p.db.Where("name = ? and path = ?", c, d).First(&folder).Error
		if err != nil {
			return o, fmt.Errorf("error getting %s %s from DB: %w", d, c, err)
		}
		// ID becomes > 0 if object found in DB
		if folder.ID > 0 {
			o.ID = folder.ID
			o.Hash = folder.Hash
			o.Name = folder.Name
			o.Path = folder.Path
			o.Owner = folder.Owner
			o.Size = folder.Size
			o.Items = folder.Items
			o.FSUpdatedAt = folder.FSUpdatedAt
			o.CreatedAt = folder.CreatedAt
			o.UpdatedAt = folder.UpdatedAt
			o.Events = []FSEvent{{Type: EventFromProto(e.Action), At: time.Now()}}
		}
	} else if e.Object.ObjectType == proto.OBJ_FILE {

		err := p.db.Where("name = ? and path = ?", c, d).First(&file).Error
		if err != nil {
			return o, fmt.Errorf("error getting %s %s from DB: %w", d, c, err)
		}
		// ID becomes > 0 if object found in DB
		if file.ID > 0 {
			o.ID = file.ID
			o.Hash = file.Hash
			o.Name = file.Name
			o.Path = file.Path
			o.Owner = file.ID
			o.Size = file.Size
			o.Ext = file.Ext
			o.FSUpdatedAt = file.FSUpdatedAt
			o.CreatedAt = file.CreatedAt
			o.UpdatedAt = file.UpdatedAt
			o.Events = []FSEvent{{Type: EventFromProto(e.Action), At: time.Now()}}
		}
	} else {
		//Even if it's an unknown oType, we still can process some data out of it

		o.FSUpdatedAt = e.Object.UpdatedAt
		o.Ext = e.Object.Ext
		o.Size = e.Object.Size
		o.Hash = e.Object.Hash
		o.Name = c
		o.Path = d
	}

	return o, nil
}

func EventFromProto(t proto.FSEventType) FSEventType {
	if t == proto.FS_ANY_ACTION {
		return FS_ANY_ACTION
	} else if t == proto.FS_CREATED {
		return FS_CREATED
	} else if t == proto.FS_DELETED {
		return FS_DELETED
	} else if t == proto.FS_UPDATED {
		return FS_UPDATED
	} else if t == proto.FS_RENAMED {
		return FS_RENAMED
	}
	return FS_UNKNOWN_ACTION
}

func (p *Fileprocessor) ConvertIntoProtoEvent(f FSObject) proto.FSEvent {
	var t proto.FSEventType
	if len(f.Events) > 1 {
		t = f.Events[len(f.Events)-1].Type.Proto()
	} else if len(f.Events) == 1 {
		t = f.Events[0].Type.Proto()
	}
	return proto.FSEvent{
		Action: t,
		Object: proto.FSObject{
			ObjectType: f.Type.Proto(),
			// Client should not be aware of its user id
			// And must treat all synced events like root dir is the only on the server
			FullPath:  p.EscapeAddress(filepath.Join(f.Path, f.Name)),
			Hash:      f.Hash,
			Ext:       f.Ext,
			Size:      f.Size,
			UpdatedAt: f.FSUpdatedAt,
		},
	}
}

func EventFromFSnotify(e fsnotify.Event) FSEventType {
	if e.Op == fsnotify.Create {
		return FS_CREATED
	} else if e.Op == fsnotify.Remove {
		return FS_DELETED
	} else if e.Op == fsnotify.Rename {
		return FS_RENAMED
	} else if e.Op == fsnotify.Write {
		return FS_UPDATED
	}
	return FS_UNKNOWN_ACTION
}
