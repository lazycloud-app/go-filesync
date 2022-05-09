package server

import (
	"fmt"

	"github.com/lazycloud-app/go-filesync/v1/basic/cloud/events"
)

// FilesystemWatcherRoutine tracks changes in every folder in root dir
func (s *Server) FilesystemWatcherRoutine() {
	s.SendVerbose(EventType("green"), events.SourceFileSystemWatcher.String(), "Starting filesystem watcher")
	done := make(chan bool)
	go func() {
		defer close(done)

		for {
			select {
			case event, ok := <-s.watcher.Events:
				if !ok {
					return
				}
				s.fsEventsChan <- event
			case err, ok := <-s.watcher.Errors:
				if !ok {
					return
				}
				s.Send(EventType("error"), events.SourceFileSystemWatcher.String(), fmt.Errorf("fs watcher error: %w", err))
			}
		}
	}()

	err := s.watcher.Add(s.config.FileSystemRootPath)
	if err != nil {
		s.Send(EventType("fatal"), events.SourceFileSystemWatcher.String(), fmt.Errorf("fs watcher add failed: %w", err))
	}
	s.SendVerbose(EventType("info"), events.SourceFileSystemWatcher.String(), fmt.Sprintf("%s added to watcher", s.config.FileSystemRootPath))
	<-done
	s.SendVerbose(EventType("red"), events.SourceFileSystemWatcher.String(), "FS watcher stopped")
}
