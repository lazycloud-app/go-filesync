package server

import (
	"fmt"

	"github.com/lazycloud-app/go-filesync/v1/basic/fs"
	"github.com/lazycloud-app/go-filesync/v1/basic/proto"
)

func (s *Server) NotificationPublisher() {
	for {
		select {
		case e := <-s.syncEventsChan:
			//If conversion result says that this action type is not for exchanging with other parties - just skip
			//It is useful in case action series (ex.: FS_RENAME->FS_CREATE)
			if e.Proto.Action == proto.FS_NO_ACTION {
				continue
			}
			//Owner = 0 means that object in server's root dir and is not meant for sync
			if e.FS.Owner == 0 {
				continue
			}
			//Client should not be aware of its user id
			// And must treat all synced events like its root dir is the only on the server
			e.Proto.Object.FullPath = fs.ExtractUser(e.Proto.Object.FullPath, e.FS.Owner)
			//Notify clients about the Event
			fmt.Println(e)
			for _, c := range s.pool.pool {
				if !c.syncActive || c.uid != e.FS.Owner {
					continue
				}
				c.eventsChan <- e.Proto
			}
		case e := <-s.syncErrChan:
			s.Error(fmt.Errorf("error in filesystem watcher: %w", e))
		}
	}
}

/*
//FilesystemWatcherRoutine reports every filesystem event that occurs within target directory (not with the directory itself)
//
//Current verion is a wrap around github.com/fsnotify and can handle create, remove, write (update) and rename events.
//Main purpose of FilesystemWatcherRoutine is to process all events by updating info in database and pass on the info
//to routine which notfies clients about changes.
func (s *Server) FilesystemWatcherRoutine() {
	s.InfoGreen("Starting filesystem watcher")
	w := s.fp.Watcher
	done := make(chan bool)
	go func() {
		defer close(done)

		for {
			select {
			case event, ok := <-w.Events:
				if !ok {
					s.Error(fmt.Errorf("[FilesystemWatcherRoutine] 'Events' channel closed"))
					return
				}
				//Process FS-event internally to the server database
				o, err := s.fp.FSEventProcess(event)
				if err != nil {
					s.Error(fmt.Errorf("[FilesystemWatcherRoutine] error processing event, %w", err))
					continue
				}
				//fmt.Println(o)
				//Add object to watch list if it was a new dir
				if o.Type == fs.OBJ_DIR && event.Op == fsnotify.Create {
					err := s.fp.Watcher.Add(event.Name)
					if err != nil {
						s.Error(fmt.Errorf("[FilesystemWatcherRoutine] fs watcher add failed: %w", err))
					}
				}

				//Wrap object & event data into correct proto.Event format
				e := s.fp.ConvertIntoProtoEvent(o)
				//If conversion result says that this action type is not for exchanging with other parties - just skip
				//It is useful in case action series (ex.: FS_RENAME->FS_CREATE)
				if e.Action == proto.FS_NO_ACTION {
					continue
				}

				//Client should not be aware of its user id
				// And must treat all synced events like its root dir is the only on the server
				e.Object.FullPath = fs.ExtractUser(e.Object.FullPath, o.Owner)
				//Notify clients about the Event
				fmt.Println(o)
				for _, c := range s.pool.pool {
					if !c.syncActive || c.uid != o.Owner {
						continue
					}
					c.eventsChan <- e
					fmt.Println("SENT to", c.uid)
				}
			case err, ok := <-w.Errors:
				if !ok {
					s.Error(fmt.Errorf("[FilesystemWatcherRoutine] 'Errors' channel closed"))
					return
				}
				s.Error(fmt.Errorf("[FilesystemWatcherRoutine] fs watcher error: %w", err))
			}
		}
	}()

	err := w.Add(s.config.FileSystemRootPath)
	if err != nil {
		s.Fatal(fmt.Errorf("fs watcher add failed: %w", err))
	}
	<-done
	s.InfoRed("FS watcher stopped")
}
*/
