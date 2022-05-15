package fs

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
)

//FilesystemWatcherRoutine reports every filesystem event that occurs within target directory (not with the directory itself)
//
//Current verion is a wrap around github.com/fsnotify and can handle create, remove, write (update) and rename events.
//Main purpose of FilesystemWatcherRoutine is to process all events by updating info in database and pass on the info
//to routine which notfies clients about changes.
func (p *Fileprocessor) FilesystemWatcherRoutine(evChan chan (EventArray), errChan chan (error)) {
	done := make(chan bool)
	go func() {
		defer close(done)

		for {
			select {
			case event, ok := <-p.Watcher.Events:
				if !ok {
					errChan <- fmt.Errorf("[FilesystemWatcherRoutine] 'Events' channel closed")
					return
				}
				if event.Name == p.root {
					fmt.Println("SKIPPED ROOT")
					continue
				}
				//Skip event if buffer says so
				for _, ba := range p.actionBuffer[event.Name] {
					if ba.Action == EventFromFSnotify(event) && ba.Ignore {
						continue
					}
				}
				//Process FS-event internally to the database
				o, err := p.FSEventProcess(event)
				if err != nil {
					errChan <- fmt.Errorf("[FilesystemWatcherRoutine] error processing event, %w", err)
					continue
				}
				fmt.Println(p.actionBuffer)
				//Add object to watch list if it was a new dir
				if o.Type == OBJ_DIR && event.Op == fsnotify.Create {
					err := p.Watcher.Add(event.Name)
					if err != nil {
						errChan <- fmt.Errorf("[FilesystemWatcherRoutine] fs watcher add failed: %w", err)
					}
				}
				//Wrap object & event data into correct proto.Event format
				e := p.ConvertIntoProtoEvent(o)
				//Notify external rutine
				evChan <- EventArray{Proto: e, FS: o}
			case err, ok := <-p.Watcher.Errors:
				if !ok {
					errChan <- fmt.Errorf("[FilesystemWatcherRoutine] 'Errors' channel closed")
					return
				}
				errChan <- fmt.Errorf("[FilesystemWatcherRoutine] fs watcher error: %w", err)
			}
		}
	}()

	err := p.Watcher.Add(p.root)
	if err != nil {
		errChan <- fmt.Errorf("fs watcher add failed: %w", err)
	}
	<-done
}
