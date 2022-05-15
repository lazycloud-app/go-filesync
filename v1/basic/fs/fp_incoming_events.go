package fs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lazybark/go-helpers/hasher"
	"github.com/lazycloud-app/go-filesync/v1/basic/proto"
	"gorm.io/gorm"
)

//FSEventProcessIncoming processes incoming event data and makes changes to database & filesystem.
//If a new object must be downloaded from event source (e.g. new version of the file), it returns
//object in proto.GetFile format which could be directly sent to peer
func (p *Fileprocessor) FSEventProcessIncoming(e proto.FSEvent) (proto.GetFile, error) {
	var gf proto.GetFile
	o, err := p.ConvertFromProtoEvent(e)
	if err != nil {
		return gf, fmt.Errorf("[FSEventProcessIncoming] error converting event: %w", err)
	}
	if e.Action == proto.FS_DELETED {

		//Putting event in buffer to ingnore in future
		p.AddEventIntoBuffer(p.UnEscapeAddress(e.Object.FullPath), FS_DELETED, true)
		//And ignore parent dir update
		p.AddEventIntoBuffer(p.UnEscapeAddress(o.Path), FS_UPDATED, true)
		return gf, p.FSProcessDeleteIncoming(o)

	} else if e.Action == proto.FS_CREATED {

		//Putting event in buffer to ingnore in future
		p.AddEventIntoBuffer(p.UnEscapeAddress(e.Object.FullPath), FS_CREATED, true)
		//And ignore parent dir update
		p.AddEventIntoBuffer(p.UnEscapeAddress(o.Path), FS_UPDATED, true)
		return p.FSProcessCreateIncoming(o)

	} else if e.Action == proto.FS_UPDATED {

		p.AddEventIntoBuffer(p.UnEscapeAddress(e.Object.FullPath), FS_UPDATED, true)
		//And ignore parent dir update
		p.AddEventIntoBuffer(p.UnEscapeAddress(o.Path), FS_UPDATED, true)
		return p.FSProcessUpdateIncoming(o)

	} else if e.Action == proto.FS_RENAMED {
		return gf, nil
	} else if e.Action == proto.FS_ANY_ACTION {
		return gf, nil
	} else if e.Action == proto.FS_NO_ACTION {
		return gf, nil
	} else if e.Action == proto.FS_UNKNOWN_ACTION {
		return gf, fmt.Errorf("[Fileprocessor -> FSEventProcessIncoming] unknown event type")
	}
	return gf, fmt.Errorf("[Fileprocessor -> FSEventProcessIncoming] unknown event type")
}

func (p *Fileprocessor) FSProcessUpdateIncoming(o FSObject) (proto.GetFile, error) {
	var gf proto.GetFile
	path := p.UnEscapeAddress(filepath.Join(o.Path, o.Name))
	if o.Type == OBJ_DIR {
		// Dir only needs updating it's update time
		// As far as files events will be sent separately
		err := os.Chtimes(path, o.FSUpdatedAt, o.FSUpdatedAt)
		if err != nil {
			return gf, fmt.Errorf("[FSProcessUpdateIncoming] error changing times '%s': %w", path, err)
		}
		// Update data in DB
		var folder Folder
		if err := p.db.Where("name = ? and path = ?", o.Name, o.Path).First(&folder).Error; err != nil {
			return gf, fmt.Errorf("[FSProcessUpdateIncoming] folder reading failed '%s': %w", path, err)
		} else {
			folder.FSUpdatedAt = o.FSUpdatedAt
			if err := p.db.Table("folders").Save(&folder).Error; err != nil && err != gorm.ErrEmptySlice {
				return gf, fmt.Errorf("[FSProcessUpdateIncoming] folder saving failed '%s': %w", path, err)
			}
		}
	} else if o.Type == OBJ_FILE {
		// File should be donwloaded only in case different hash value
		var file File
		if err := p.db.Where("name = ? and path = ?", o.Name, o.Path).First(&file).Error; err != nil {

			return gf, fmt.Errorf("[FSProcessUpdateIncoming] dir saving failed '%s': %w", path, err)

		} else {
			// Update data in DB
			hash := ""
			hash, err := hasher.HashFilePath(path, hasher.SHA256, 8192)
			if err != nil {
				return gf, fmt.Errorf("[FSProcessUpdateIncoming] error hashing file '%s': %w", path, err)
			}
			if hash == o.Hash {
				file.Hash = hash
				file.FSUpdatedAt = o.FSUpdatedAt
				if err := p.db.Table("files").Save(&file).Error; err != nil && err != gorm.ErrEmptySlice {
					return gf, fmt.Errorf("[FSProcessUpdateIncoming] error hashing file '%s': %w", path, err)
				}
			} else {
				gf = proto.GetFile{
					Name:      o.Name,
					Path:      o.Path,
					Hash:      o.Hash,
					UpdatedAt: o.FSUpdatedAt,
				}
				return gf, nil
			}
		}
	}
	return gf, nil
}

func (p *Fileprocessor) FSProcessCreateIncoming(o FSObject) (proto.GetFile, error) {
	var gf proto.GetFile
	path := p.UnEscapeAddress(filepath.Join(o.Path, o.Name))
	if o.Type == OBJ_DIR {
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return gf, fmt.Errorf("[ProcessObjectCreated] error making path '%s': %w", path, err)
		}
		err := os.Chtimes(path, o.UpdatedAt, o.UpdatedAt)
		if err != nil {
			return gf, fmt.Errorf("[ProcessObjectCreated] error changing times '%s': %w", path, err)
		}

		err = p.Watcher.Add(path)
		if err != nil {
			return gf, fmt.Errorf("[ProcessObjectCreated] error adding to watcher '%s': %w", path, err)
		}

		// Scan dir
		_, _, err, errs := p.ProcessDirectory(path)
		if err != nil {
			return gf, fmt.Errorf("[ProcessObjectCreated] processing dir '%s': %w", path, err)
		}
		if len(errs) > 0 {
			return gf, ProcessingError{Type: ERR_DIRECTORY_PROCESSING, ErrList: errs}
		}

		dInfo, err := os.Stat(path)
		if err != nil {
			return gf, fmt.Errorf("[ProcessObjectCreated] object reading failed '%s': %w", path, err)
		}

		folder := Folder{
			Name:        o.Name,
			FSUpdatedAt: dInfo.ModTime(),
			Path:        o.Path,
			Size:        dInfo.Size(),
		}

		err = p.RecordDir(folder)
		if err != nil && err.Error() != "UNIQUE constraint failed: files.name, files.path" {
			return gf, fmt.Errorf("error making record for %s: %w", path, err)
		}
		/*****************************************************/
		/*****************************************************/
		/*****************************************************/
		/*****************************************************/
		/* And here we will ask server to send full list of directory files in case in was crated not empty*/
		/*****************************************************/
		/*****************************************************/
		/*****************************************************/

	} else if o.Type == OBJ_FILE {
		gf = proto.GetFile{
			Name:      o.Name,
			Path:      o.Path,
			Hash:      o.Hash,
			UpdatedAt: o.UpdatedAt,
		}
		return gf, nil
	}
	return gf, nil
}

func (p *Fileprocessor) FSProcessDeleteIncoming(o FSObject) error {
	path := p.UnEscapeAddress(filepath.Join(o.Path, o.Name))
	oData, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("object reading failed '%s': %w", path, err)
	}
	if oData.ModTime().After(o.FSUpdatedAt) {
		return ProcessingError{Type: ERR_NEWER_VERSION_EXISTS}
	}

	var folder Folder
	var file File
	if o.Type == OBJ_DIR {
		err = p.db.Where("name = ? and path = ?", o.Name, o.Path).First(&folder).Error
		if err != nil {
			return fmt.Errorf("object DB reading failed '%s': %w", path, err)
		}
		err = p.db.Delete(&folder).Error
		if err != nil {
			return fmt.Errorf("object DB deleting failed '%s': %w", path, err)
		}
		// Manually delete all files connected to this dir
		err = p.db.Where("path = ?", o.Path).Delete(&file).Error
		if err != nil {
			return fmt.Errorf("object DB deleting related files failed '%s': %w", path, err)
		}
		err = os.RemoveAll(path)
		if err != nil {
			return fmt.Errorf("error removing '%s': %w", path, err)
		}
	} else if o.Type == OBJ_FILE {
		err := p.db.Where("name = ? and path = ?", o.Name, o.Path).First(&file).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return fmt.Errorf("object DB reading failed '%s': %w", path, err)
		}
		err = p.db.Delete(&file).Error
		if err != nil {
			return fmt.Errorf("object DB deleting failed '%s': %w", path, err)
		}
		err = os.RemoveAll(path)
		if err != nil {
			return fmt.Errorf("[RemoveAll] error removing '%s': %w", path, err)
		}
	} else {
		//Even if object is unknown, we still can try to perform some actions
	}

	return nil
}
