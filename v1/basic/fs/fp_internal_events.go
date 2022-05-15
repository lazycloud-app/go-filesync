package fs

import (
	"fmt"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/lazybark/go-helpers/hasher"
	"gorm.io/gorm"
)

func (s *Fileprocessor) FSEventProcess(e fsnotify.Event) (o FSObject, err error) {
	if e.Op == fsnotify.Remove {
		return s.FSProcessDelete(e.Name)
	} else if e.Op == fsnotify.Write {
		return s.FSProcessUpdate(e.Name)
	} else if e.Op == fsnotify.Rename {
		return s.FSProcessRename(e.Name)
	} else if e.Op == fsnotify.Create {
		return s.FSProcessCreate(e.Name)
	}
	return o, fmt.Errorf("[Fileprocessor -> FSEventProcess] unknown event type")
}

func (f *Fileprocessor) FSProcessUpdate(a string) (r FSObject, err error) {
	dir, child := SplitPath(a)

	oInfo, err := os.Stat(a)
	if err != nil {
		return r, fmt.Errorf("[FSProcessUpdate] object %s reading failed: %w", a, err)
	}

	if oInfo.IsDir() {
		var folder Folder

		if err := f.db.Where("name = ? and path = ?", child, f.EscapeAddress(dir)).First(&folder).Error; err != nil {

			return r, fmt.Errorf("[FSProcessUpdate] dir reading failed: %w", err)
		} else {
			// Update data in DB
			folder.FSUpdatedAt = oInfo.ModTime()
			if err := f.db.Table("folders").Save(&folder).Error; err != nil && err != gorm.ErrEmptySlice {
				return r, fmt.Errorf("[FSProcessUpdate] dir saving failed: %w", err)
			}
		}
		r = FSObject{
			Type:        OBJ_DIR,
			ID:          folder.ID,
			Hash:        "",
			Name:        folder.Name,
			Path:        folder.Path,
			Owner:       folder.Owner,
			Size:        folder.Size,
			Ext:         "",
			FSUpdatedAt: folder.FSUpdatedAt,
			CreatedAt:   folder.CreatedAt,
			UpdatedAt:   folder.UpdatedAt,
			Events:      []FSEvent{{Type: FS_UPDATED, At: time.Now()}},
		}
		return r, nil
	} else {
		var file File
		if err := f.db.Where("name = ? and path = ?", child, f.EscapeAddress(dir)).First(&file).Error; err != nil {
			/*if err == gorm.ErrRecordNotFound {
				fmt.Println("SHIIIIT ==")
			}
			if errors.As(err, &gorm.ErrRecordNotFound) {
				fmt.Println("SHIIIIT as")
			}
			if errors.Is(err, gorm.ErrRecordNotFound) {
				fmt.Println("SHIIIIT is==")
			}
			fmt.Println(reflect.ValueOf(err))
			fmt.Println(reflect.TypeOf(err))*/
			return r, fmt.Errorf("[FSProcessUpdate] file reading failed: %w", err)
		} else {
			// Update data in DB
			hash := ""
			hash, err := hasher.HashFilePath(a, hasher.SHA256, 8192)
			if err != nil {
				return r, fmt.Errorf("[FSProcessUpdate] error getting hash: %w", err)

			}
			file.FSUpdatedAt = oInfo.ModTime()
			file.Size = oInfo.Size()
			file.Hash = hash
			if err := f.db.Table("files").Save(&file).Error; err != nil && err != gorm.ErrEmptySlice {
				return r, fmt.Errorf("[FSProcessUpdate] file saving failed: %w", err)
			}

			r = FSObject{
				Type:        OBJ_FILE,
				ID:          file.ID,
				Hash:        file.Hash,
				Name:        file.Name,
				Path:        file.Path,
				Owner:       file.Owner,
				Size:        file.Size,
				Ext:         file.Ext,
				FSUpdatedAt: file.FSUpdatedAt,
				CreatedAt:   file.CreatedAt,
				UpdatedAt:   file.UpdatedAt,
				Events:      []FSEvent{{Type: FS_UPDATED, At: time.Now()}},
			}
			return r, nil

		}

	}
}

func (f *Fileprocessor) FSProcessCreate(a string) (r FSObject, err error) {
	oInfo, err := os.Stat(a)
	if err != nil {
		return r, fmt.Errorf("[FSProcessCreate] os.Stat failed: %w", err)
	}

	if oInfo.IsDir() {
		// Scan dir
		_, _, err, errs := f.ProcessDirectory(a)
		if err != nil {
			return r, fmt.Errorf("[FSProcessCreate] error processing %s: %w", a, err)
		}
		if len(errs) > 0 {
			return r, ProcessingError{Type: ERR_DIRECTORY_PROCESSING, ErrList: errs}
		}
		// Check dir data
		dir, err := f.ProcessFolder(oInfo, a)
		if err != nil {
			return r, fmt.Errorf("[ObjectCreated] ProcessFolder failed: %w", err)
		}

		r = FSObject{
			Type:        OBJ_DIR,
			ID:          dir.ID,
			Hash:        "",
			Name:        dir.Name,
			Path:        dir.Path,
			Owner:       dir.Owner,
			Size:        dir.Size,
			Ext:         "",
			FSUpdatedAt: dir.FSUpdatedAt,
			CreatedAt:   dir.CreatedAt,
			UpdatedAt:   dir.UpdatedAt,
			Events:      []FSEvent{{Type: FS_CREATED, At: time.Now()}},
		}

	} else {
		file, err := f.ProcessFile(oInfo, a)
		if err != nil {
			return r, fmt.Errorf("[FSProcessCreate] ProcessFile failed: %w", err)
		}
		//Check if there is a cached FS_RENAME event with the file
		var t FSEventType
		ren := File{}
		f.db.Where("hash = ? and path = ? and is_renamed = ?", file.Hash, f.EscapeAddress(file.Path), true).First(&ren)
		if ren.ID > 0 {
			fmt.Println("dsfsdf")
			t = FS_RENAMED
			if err := f.db.Delete(&ren).Error; err != nil {
				return r, fmt.Errorf("[FSProcessCreate] file renaming failed: %w", err)
			}
		} else {
			t = FS_CREATED
		}
		/*if name, ok := f.rb[file.Hash]; ok {
			t = FS_RENAMED
			//Delete from buffer
			delete(f.rb, file.Hash)
			//Save new name in DB
			dir, obj := SplitPath(name)
			if err := f.db.Where("name = ? and path = ?", obj, dir).Delete(&File{}).Error; err != nil {
				return r, fmt.Errorf("[FSProcessCreate] file renaming failed: %w", err)
			}
		} else {
			t = FS_CREATED
		}*/
		// Client should not be aware of its user id
		// And must treat all synced events like its root dir is the only on the server
		r = FSObject{
			Type:        OBJ_FILE,
			ID:          file.ID,
			Hash:        file.Hash,
			Name:        file.Name,
			Path:        file.Path,
			Owner:       file.Owner,
			Size:        file.Size,
			Ext:         file.Ext,
			FSUpdatedAt: file.FSUpdatedAt,
			CreatedAt:   file.CreatedAt,
			UpdatedAt:   file.UpdatedAt,
			Events:      []FSEvent{{Type: t, At: time.Now()}},
		}
	}

	return r, nil
}

func (f *Fileprocessor) FSProcessRename(a string) (r FSObject, err error) {
	var file File
	var folder Folder
	//Find dir and object names
	dir, child := SplitPath(a)
	//Find the object in database
	//Check if it was a file
	err = f.db.Where("name = ? and path = ?", child, f.EscapeAddress(dir)).First(&file).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return r, fmt.Errorf("error getting %s from DB: %w", a, err)
	}
	// ID becomes > 0 if object found in DB
	if file.ID > 0 {
		file.IsRenamed = true
		if err := f.db.Table("files").Save(&file).Error; err != nil && err != gorm.ErrEmptySlice {
			return r, fmt.Errorf("[FSProcessUpdate] file saving failed: %w", err)
		}
		return r, nil
	}

	//Or a dir
	err = f.db.Where("name = ? and path = ?", child, f.EscapeAddress(dir)).First(&folder).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return r, fmt.Errorf("error getting %s from DB: %w", a, err)
	}
	// ID becomes > 0 if object found in DB
	if folder.ID > 0 {
		folder.IsRenamed = true
		if err := f.db.Table("folders").Save(&folder).Error; err != nil && err != gorm.ErrEmptySlice {
			return r, fmt.Errorf("[FSProcessUpdate] folder saving failed: %w", err)
		}
		return r, nil
	}

	return r, fmt.Errorf("[FSProcessRename] object %s not found in database", a)
}

//FSProcessDelete provides all necessary actions to treat file/dir deletion of a,
// where a is the full path to object
func (f *Fileprocessor) FSProcessDelete(a string) (r FSObject, err error) {
	var file File
	var folder Folder
	//Find dir and object names
	dir, child := SplitPath(a)

	//Check if it was a file
	err = f.db.Where("name = ? and path = ?", child, f.EscapeAddress(dir)).First(&file).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return r, fmt.Errorf("error getting %s from DB: %w", a, err)
	}
	// ID becomes > 0 if object found in DB
	if file.ID > 0 {
		r = FSObject{
			Type:        OBJ_FILE,
			ID:          file.ID,
			Hash:        file.Hash,
			Name:        child,
			Path:        dir,
			Owner:       file.Owner,
			Size:        file.Size,
			Ext:         file.Ext,
			FSUpdatedAt: file.FSUpdatedAt,
			CreatedAt:   file.CreatedAt,
			UpdatedAt:   file.UpdatedAt,
			Events:      []FSEvent{{Type: FS_DELETED, At: time.Now()}},
		}

		err = f.db.Delete(&file).Error
		if err != nil {
			return r, fmt.Errorf("error deleting %s: %w", a, err)
		}
		return r, nil
	}

	err = f.db.Where("name = ? and path = ?", child, f.EscapeAddress(dir)).First(&folder).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return r, fmt.Errorf("error getting %s from DB: %w", a, err)
	}
	// ID becomes > 0 if object found in DB
	if folder.ID > 0 {
		err = f.db.Delete(&folder).Error
		if err != nil {
			return r, fmt.Errorf("error deleting %s: %w", a, err)
		}
		// Manually delete all files connected to this dir
		err = f.db.Where("path = ?", f.EscapeAddress(a)).Delete(&file).Error
		if err != nil {
			return r, fmt.Errorf("error deleting files associated to %s: %w", a, err)
		}

		r = FSObject{
			Type:        OBJ_DIR,
			ID:          folder.ID,
			Hash:        folder.Hash,
			Name:        child,
			Path:        dir,
			Owner:       folder.Owner,
			Size:        folder.Size,
			Items:       0,
			FSUpdatedAt: folder.FSUpdatedAt,
			CreatedAt:   folder.CreatedAt,
			UpdatedAt:   folder.UpdatedAt,
			Events:      []FSEvent{{Type: FS_DELETED, At: time.Now()}},
		}
		return r, nil
	}

	return r, nil
}
