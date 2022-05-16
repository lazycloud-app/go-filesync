package fp

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/lazycloud-app/go-filesync/v1/v1/db"
)

//FP is a FileProcessor interface implementation which controls how all objects in FS are treated.
//Also FP watches for any changes in FS and notifies main routine
type FP struct {
	//Filesystem root unrelative to host filesystem structure.
	//Should be replaced by RootPointer in all sync events
	root string
	//Database to store filesystem state.
	//Highly recommended to use db shared with main app: it will make whole system more portable.
	db db.DataBase
	//Some FP methods do not return errors. Instead warnings sent to outer routine via this channel,
	//so only client/server will decide when to stop execution.
	errChan chan (error)
	//w points to fsnotify.Watcher which is used to get filesystem events
	w *fsnotify.Watcher
}

//New creates new FP and sets all necessary variables
func New(root string, db db.DataBase, errChan chan (error)) (fp *FP, err error) {
	fp = new(FP)
	fp.SetRoot(root)

	fp.db = db
	fp.w, err = fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("[New FP] NewWatcher failed: %w", err)
	}

	fp.errChan = errChan

	return fp, nil
}

//SetRoot sets root directory for FileProcessor which will be used to escape/unescape paths
func (fp *FP) SetRoot(r string) {
	fp.root = r
}

//WatchRoot calls to Watch using root directory as the argument
func (fp *FP) WatchRoot() {
	fp.Watch(fp.root)
}

//Watch sets internal watcher to monitor filesystem changes in d. Errors are returned via fp.errChan
func (fp *FP) Watch(d string) {
	if CheckEscaped(d) {
		d = fp.UnEscapeAddress(d)
	}
	err := fp.w.Add(d)
	if err != nil {
		fp.errChan <- fmt.Errorf("[Watch] watcher for '%s' failed: %w", d, err)
	}
}
