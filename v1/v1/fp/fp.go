package fp

import (
	"strings"
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
}

//Delim is the FS-safe delimeter that should replace any other delimeter before sending filepath to any peer.
//
//Its use will reduce number of pissble delimeters to check in path and it does not need any escape in strings.
//So conversion becomes very simple by replacing 'X' to '>' to 'Y'
var Delim = ">"

func CheckEscaped(path string) bool {
	return strings.Contains(path, Delim)
}
