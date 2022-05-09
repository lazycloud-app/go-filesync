package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/lazybark/go-helpers/hasher"
	"github.com/lazycloud-app/go-filesync/v1/basic/cloud/events"
	"github.com/lazycloud-app/go-filesync/v1/basic/proto"
	"gorm.io/gorm"
)

type (
	FSEventNotification struct {
		// Event in fsnotify.Event format (with Op, Name fields and Op.String() method)
		Event fsnotify.Event
		// Filesystem object (file or directory)
		Object interface{}
	}
)

// FSEventsProcessor works with all events that come from FS watcher
func (s *Server) FSEventsProcessor() {
	s.SendVerbose(EventType("cyan"), events.SourceFsEvents.String(), "Starting FSEventsProcessor")

	for event := range s.fsEventsChan {
		var ne FSEventNotification
		owner := s.fw.GetOwner(event.Name)
		ne.Event = event

		if event.Op == fsnotify.Create {
			err := s.ObjectCreated(event.Name, &ne)
			if err != nil {
				s.Send(EventType("error"), events.SourceFsEvents.String(), fmt.Errorf("[FSEventsProcessor] %s processing failed: %w", event.Op.String(), err))
				continue
			}
		} else if event.Op == fsnotify.Remove {
			err := s.ObjectDeleted(event.Name, &ne)
			if err != nil {
				s.Send(EventType("error"), events.SourceFsEvents.String(), fmt.Errorf("[FSEventsProcessor] %s processing failed: %w", event.Op.String(), err))
				continue
			}
		} else if event.Op == fsnotify.Rename {
			continue
		} else if event.Op == fsnotify.Write {
			err := s.ObjectWrite(event.Name, &ne)
			if err != nil {
				s.Send(EventType("error"), events.SourceFsEvents.String(), fmt.Errorf("[FSEventsProcessor] %s processing failed: %w", event.Op.String(), err))
				continue
			}
		}

		// Send notification to active clients
		for _, c := range s.activeConnections {
			if !c.SyncActive || c.Uid != owner {
				continue
			}
			c.EventsChan <- ne
			fmt.Println("Event sent")
		}

		/* else if event.Op.String() == "WRITE" {

			}
		}*/

	}
	s.Send(EventType("red"), events.SourceFsEvents.String(), "FSEventsProcessor closed")
}

func (s *Server) ObjectWrite(object string, ne *FSEventNotification) error {
	dir, child := filepath.Split(object)
	dir = strings.TrimSuffix(dir, string(filepath.Separator))

	dat, err := os.Stat(object)
	if err != nil {
		return fmt.Errorf("object %s reading failed: %w", object, err)
	}
	if dat.IsDir() {
		var folder proto.Folder

		if err := s.db.Where("name = ? and path = ?", child, s.fw.EscapeAddress(dir)).First(&folder).Error; err != nil {
			return fmt.Errorf("file reading failed: %w", err)
		} else {
			// Update data in DB
			folder.FSUpdatedAt = dat.ModTime()
			if err := s.db.Table("folders").Save(&folder).Error; err != nil && err != gorm.ErrEmptySlice {
				return fmt.Errorf("dir saving failed: %w", err)
			}
		}
		folder.Path = s.fw.ExtractUser(s.fw.EscapeAddress(folder.Path), folder.Owner)
		ne.Object = folder
	} else {
		var file proto.File
		if err := s.db.Where("name = ? and path = ?", child, s.fw.EscapeAddress(dir)).First(&file).Error; err != nil {
			return fmt.Errorf("file reading failed: %w", err)
		} else {
			// Update data in DB
			hash := ""
			hash, err := hasher.HashFilePath(object, hasher.SHA256, 8192)
			if err != nil {
				return fmt.Errorf("error getting hash: %w", err)

			}
			file.FSUpdatedAt = dat.ModTime()
			file.Size = dat.Size()
			file.Hash = hash
			if err := s.db.Table("files").Save(&file).Error; err != nil && err != gorm.ErrEmptySlice {
				return fmt.Errorf("file saving failed: %w", err)
			}

		}
		file.Path = s.fw.ExtractUser(file.Path, file.Owner)
		ne.Object = file
	}
	return nil
}

func (s *Server) ObjectDeleted(object string, ne *FSEventNotification) error {
	var file proto.File
	var folder proto.Folder

	dir, child := filepath.Split(object)
	dir = strings.TrimSuffix(dir, string(filepath.Separator))

	err := s.db.Where("name = ? and path = ?", child, s.fw.EscapeAddress(dir)).First(&file).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("error getting %s from DB: %w", dir+child, err)
	}
	// ID becomes real if object found in DB
	if file.ID > 0 {
		err = s.db.Delete(&file).Error
		if err != nil {
			return fmt.Errorf("error deleting %s: %w", dir+child, err)
		}
		file.Path = s.fw.ExtractUser(file.Path, file.Owner)
		ne.Object = file

	}

	err = s.db.Where("name = ? and path = ?", child, s.fw.EscapeAddress(dir)).First(&folder).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("error getting %s from DB: %w", dir+child, err)
	}
	// ID becomes real if object found in DB
	if folder.ID > 0 {
		err = s.db.Delete(&folder).Error
		if err != nil {
			return fmt.Errorf("error deleting %s: %w", object, err)
		}
		// Manually delete all files connected to this dir
		err = s.db.Where("path = ?", s.fw.EscapeAddress(object)).Delete(&file).Error
		if err != nil {
			return fmt.Errorf("error deleting files assiciated to %s: %w", object, err)
		}

		folder.Path = s.fw.ExtractUser(s.fw.EscapeAddress(folder.Path), folder.Owner)
		folder.FSUpdatedAt = time.Now()
		ne.Object = folder
	}

	return nil
}

func (s *Server) ObjectCreated(object string, ne *FSEventNotification) error {
	oInfo, err := s.fw.ScanObject(object)
	if err != nil {
		return fmt.Errorf("[ObjectCreated] ScanObject failed: %w", err)

	}

	if oInfo.IsDir() {
		// Watch new dir
		err := s.watcher.Add(object)
		if err != nil {
			return fmt.Errorf("[ObjectCreated] fs watcher add failed: %w", err)
		}

		// Scan dir
		_, _, err, errs := s.fw.ProcessDirectory(object)
		if err != nil {
			return fmt.Errorf("error processing %s: %w", object, err)
		}
		if len(errs) > 0 {
			for _, v := range errs {
				fmt.Println(v)
			}
		}
		// Check dir data
		dir, err := s.fw.ProcessFolder(oInfo, object)
		if err != nil {
			return fmt.Errorf("[ObjectCreated] ProcessFolder failed: %w", err)
		}
		// Client should not be aware of its user id
		// And must treat all synced events like its root dir is the only on the server
		dir.Path = s.fw.ExtractUser(dir.Path, dir.Owner)
		ne.Object = dir

	} else {
		file, err := s.fw.ProcessFile(oInfo, object)
		if err != nil {
			return fmt.Errorf("[ObjectCreated] ProcessFile failed: %w", err)
		}
		// Client should not be aware of its user id
		// And must treat all synced events like its root dir is the only on the server
		file.Path = s.fw.ExtractUser(file.Path, file.Owner)
		ne.Object = file
	}

	return nil
}
