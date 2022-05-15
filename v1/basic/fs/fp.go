package fs

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/lazybark/go-helpers/hasher"
	"gorm.io/gorm"
)

func NewProcessor(root string, w *fsnotify.Watcher, db *gorm.DB) *Fileprocessor {
	f := new(Fileprocessor)
	f.root = root
	f.Watcher = w
	f.db = db
	f.actionBuffer = make(map[string][]BufferedAction)
	f.abMutex = &sync.RWMutex{}
	return f
}

func (f *Fileprocessor) ProcessDirectory(path string) (files int, dirs int, err error, processErrors []error) {
	var dirsArray []Folder
	var filesArray []File
	unescaped := f.UnEscapeAddress(path)

	ok := f.CheckPathConsistency(unescaped)
	if !ok {
		return files, dirs, fmt.Errorf("provided path is not valid or consistent"), processErrors
	}

	errs := f.ScanDir(unescaped, &dirsArray, &filesArray)
	if len(errs) > 0 {
		processErrors = append(processErrors, errs...)
	}

	var dirPath string
	for _, dir := range dirsArray {
		dirPath = f.UnEscapeAddress(filepath.Join(dir.Path, dir.Name))
		err = f.Watcher.Add(dirPath)
		if err != nil {
			processErrors = append(processErrors, err)
			return
		}
	}

	files = len(filesArray)
	dirs = len(dirsArray)

	return
}

func (f *Fileprocessor) ScanDir(path string, dirs *[]Folder, files *[]File) (errorsScan []error) {
	var dirsRec []string
	var dir Folder
	var file File

	contents, err := ioutil.ReadDir(path)
	if err != nil {
		return
	}

	for _, item := range contents {
		p := filepath.Join(path, item.Name())
		if item.IsDir() {
			// Send dirname for recursive scanning
			dirsRec = append(dirsRec, item.Name())
			// Append to dirs list for output
			dir, err = f.ProcessFolder(item, p)
			if err != nil {
				errorsScan = append(errorsScan, fmt.Errorf("[ScanDir] error processing folder %s: %w", p, err))
			}
			*dirs = append(*dirs, dir)
		} else {
			file, err = f.ProcessFile(item, p)
			if err != nil {
				errorsScan = append(errorsScan, fmt.Errorf("[ScanDir] error processing file %s: %w The file will be temporarily out of sync", p, err))
			}
			// Append to list for output
			*files = append(*files, file)
		}
	}

	// Recursively scan all sub dirs
	for _, dirName := range dirsRec {
		errs := f.ScanDir(filepath.Join(path, dirName), dirs, files)
		if len(errs) > 0 {
			errorsScan = append(errorsScan, errs...)
		}
	}

	return
}

func (f *Fileprocessor) ProcessFolder(oInfo fs.FileInfo, fullPath string) (folder Folder, err error) {
	if !oInfo.IsDir() {
		return folder, fmt.Errorf("[ProcessFolder] provided object is NOT a dir")
	}

	dir, _ := SplitPath(fullPath)

	folder.Name = oInfo.Name()
	folder.FSUpdatedAt = oInfo.ModTime()
	folder.Path = f.EscapeAddress(dir)
	folder.Size = oInfo.Size()
	folder.FSUpdatedAt = oInfo.ModTime()
	folder.Owner = f.GetOwner(dir)

	err = f.RecordDir(folder)
	if err != nil && err.Error() != "UNIQUE constraint failed: files.name, files.path" {
		return folder, fmt.Errorf("error making record for %s: %w", fullPath, err)
	}

	return
}

func (f *Fileprocessor) GetOwner(path string) uint {
	// ownerId will extract user id that owns directory in root
	var ownerId = regexp.MustCompile(`%ROOT_DIR%,(\d*)`)
	// Only work with escaped addresses to avoid ambiguity
	if !f.CheckEscaped(path) {
		path = f.EscapeAddress(path)
	}
	user := ownerId.FindStringSubmatch(path)

	if len(user) < 2 {
		return 0
	}

	userReal, err := strconv.Atoi(user[1])
	if err != nil {
		return 0
	}

	return uint(userReal)
}

func (f *Fileprocessor) RecordFile(record File) error {
	if err := f.db.Model(&File{}).Save(&record).Error; err != nil && err != gorm.ErrEmptySlice {
		return err
	}
	return nil
}

func (f *Fileprocessor) RecordDir(record Folder) error {
	if err := f.db.Model(&Folder{}).Save(&record).Error; err != nil && err != gorm.ErrEmptySlice {
		return err
	}
	return nil
}

func (f *Fileprocessor) ProcessFile(oInfo fs.FileInfo, fullPath string) (file File, err error) {
	if oInfo.IsDir() {
		return file, fmt.Errorf("[ProcessFile] provided object is a dir")
	}
	var hash string
	dir, _ := filepath.Split(fullPath)
	dir = strings.TrimSuffix(dir, string(filepath.Separator))

	// retry lets us avoid most stupid errors, like "the previous app didn't release it as fast as we would like"
	retry := 0
	var errHashing error
	for {
		hash, errHashing = hasher.HashFilePath(fullPath, hasher.SHA256, 8192)
		if errHashing == nil {
			break
		} else if errHashing != nil && retry < 15 {
			retry++
			time.Sleep(1 * time.Second)
		} else {
			// break in all cases, but return an error
			errHashing = fmt.Errorf("error getting hash: %w", err)
			break
		}
	}

	file.Name = oInfo.Name()
	file.FSUpdatedAt = oInfo.ModTime()
	file.Path = f.EscapeAddress(dir)
	file.Hash = hash
	file.Size = oInfo.Size()
	file.FSUpdatedAt = oInfo.ModTime()
	file.Ext = filepath.Ext(oInfo.Name())
	file.Owner = f.GetOwner(dir)

	err = f.RecordFile(file)
	if err != nil && err.Error() != "UNIQUE constraint failed: files.name, files.path" {
		return file, fmt.Errorf("error making record for %s: %w", fullPath, err)
	}

	// We return errHashing in case file was busy - it will be out of sync for some time
	return file, errHashing
}
