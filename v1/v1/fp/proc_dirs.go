package fp

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"path/filepath"

	"github.com/lazycloud-app/go-filesync/v1/v1/md"
)

//ProcessDirectoryToDB implements FileProcessor interface. Method scans and saves into DB all available content in specified dir.
//It does not save specified dir data itself, only ones that included.
func (fp *FP) ProcessDirectoryToDB(path string) (files, dirs int, err error) {
	var filesArray []md.File
	var dirsArray []md.Folder

	//In case addres was escaped and has not right path delimeters for current filesystem
	if CheckEscaped(path) {
		path = fp.UnEscapeAddress(path)
	}
	//We need to be sure that there will be no sudden doubles in DB, so path to an object should always follow same rules
	ok := fp.CheckPathConsistency(path)
	if !ok {
		return files, dirs, fmt.Errorf("[ProcessDirectoryToDB] provided path is not valid or consistent")
	}

	//Recursively scan all internal contents
	fp.ScanDir(path, &dirsArray, &filesArray)

	//Save dirs & files found in provided path
	err = fp.db.RecordDir(dirsArray)
	if err != nil {
		return files, dirs, fmt.Errorf("[ProcessDirectoryToDB] error savig dirs list: %w", err)
	}
	err = fp.db.RecordFile(filesArray)
	if err != nil {
		return files, dirs, fmt.Errorf("[ProcessDirectoryToDB] error savig files list: %w", err)
	}

	//Now watch all subdirs
	for _, d := range dirsArray {
		fp.Watch(filepath.Join(d.Path, d.Name))
	}

	files = len(filesArray)
	dirs = len(dirsArray)
	return files, dirs, nil
}

//ScanDir reads dir properties and contents. All included dirs will be recursively scanned with ScanDir
//
//Method does not return errors - all will be sent to fp.errChan
func (fp *FP) ScanDir(path string, dirs *[]md.Folder, files *[]md.File) {
	var dirsRec []string
	var dir md.Folder
	var file md.File

	//Get all available dir content
	contents, err := ioutil.ReadDir(path)
	if err != nil {
		fp.errChan <- fmt.Errorf("[ScanDir] error reading directory data for %s: %w", path, err)
		return
	}

	for _, item := range contents {
		p := filepath.Join(path, item.Name())
		if item.IsDir() {
			//Send dir full path for recursive scanning
			dirsRec = append(dirsRec, item.Name())
			dir, err = fp.ProcessDirectory(item, p)
			if err != nil {
				fp.errChan <- fmt.Errorf("[ScanDir] error processing directory %s: %w", p, err)
			}
			//Append to list for output
			*dirs = append(*dirs, dir)
		} else {
			file, err = fp.ProcessFile(item, p)
			if err != nil {
				fp.errChan <- fmt.Errorf("[ScanDir] error processing file %s: %w", p, err)
			}
			//Append to list for output
			*files = append(*files, file)
		}
	}

	// Recursively scan all sub dirs
	for _, dirName := range dirsRec {
		fp.ScanDir(filepath.Join(path, dirName), dirs, files)
	}
}

func (fp *FP) ProcessDirectory(oInfo fs.FileInfo, path string) (folder md.Folder, err error) {
	if !oInfo.IsDir() {
		return folder, fmt.Errorf("[ProcessFolder] provided object is NOT a dir")
	}

	dir, _ := fp.SplitPath(path)

	folder.Name = oInfo.Name()
	folder.FSUpdatedAt = oInfo.ModTime()
	folder.Path = fp.EscapeAddress(dir)
	folder.Size = oInfo.Size()
	folder.FSUpdatedAt = oInfo.ModTime()
	folder.Owner = fp.GetOwner(dir)
	folder.Hash = ""

	return folder, nil
}
