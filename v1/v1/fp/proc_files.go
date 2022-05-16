package fp

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/lazybark/go-helpers/hasher"
	"github.com/lazycloud-app/go-filesync/v1/v1/md"
)

func (fp *FP) ProcessFile(oInfo fs.FileInfo, path string) (file md.File, err error) {
	if oInfo.IsDir() {
		return file, fmt.Errorf("[ProcessFile] provided object is a dir")
	}
	var hash string
	dir, _ := filepath.Split(path)
	dir = strings.TrimSuffix(dir, string(filepath.Separator))

	//Retry lets us avoid most stupid error cases, like "the previous app didn't release the file as fast as we would like"
	retry := 0
	var errHashing error
	for {
		hash, errHashing = hasher.HashFilePath(path, hasher.SHA256, 8192)
		if errHashing == nil {
			break
		} else if errHashing != nil && retry < 15 {
			retry++
			time.Sleep(1 * time.Second)
		} else {
			//Break in rest of cases, but return an error. File will be processed except its hash.
			//Protocol will not allow sync the file, so hash needs to be calculated later
			errHashing = fmt.Errorf("error getting hash: %w", err)
			break
		}
	}

	file.Name = oInfo.Name()
	file.FSUpdatedAt = oInfo.ModTime()
	file.Path = fp.EscapeAddress(dir)
	file.Hash = hash
	file.Size = oInfo.Size()
	file.FSUpdatedAt = oInfo.ModTime()
	file.Ext = filepath.Ext(oInfo.Name())
	file.Owner = fp.GetOwner(dir)

	// We return errHashing in case file was busy
	return file, errHashing
}
