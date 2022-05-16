package fp

import (
	"strings"

	p "github.com/lazycloud-app/go-fsp-proto/fileprocessing"
)

//FileProcessor is the general interface to process filesystem, including all kinds of events.
type FileProcessor interface {
	//SetRoot points to root directory of server/client.
	SetRoot(string)
	//ProcessDirectoryToDB meant to scan directory contents and put all found objects to database (but except the dir itself).
	//How exactly this happens and where will be stored depends on specific interface implementation.
	//
	//It returns number of directories and files into processed path.
	ProcessDirectoryToDB(string) (int, int, error)
	//WatchRoot sets FileProcessor internal watcher to monitor its root directory for any filesystem changes
	WatchRoot()
	//Watch sets FileProcessor internal watcher to monitor specified directory for any filesystem changes
	Watch(string)
}

//Delim is the FS-safe delimeter that should replace any other delimeter before sending filepath to any peer.
//
//Its use will reduce number of possible delimeters to check in path and it does not need any escape in strings.
//So conversion becomes very simple by replacing 'X' to Delim to 'Y'
var Delim = p.Delim

//RootPointer is the text representation of root path in FS-safe way
var RootPointer = p.RootPointer

//CheckEscaped returns true if finds Delim or RootPointer in provided path
func CheckEscaped(path string) bool {
	if strings.Contains(path, Delim) {
		return true
	}
	if strings.Contains(path, RootPointer) {
		return true
	}
	return false
}
