package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (f *Fileprocessor) EscapeAddress(address string) string {
	address = strings.ReplaceAll(address, f.root, "%ROOT_DIR%")
	return strings.ReplaceAll(address, string(filepath.Separator), ",")
}

func (f *Fileprocessor) UnEscapeAddress(address string) string {
	address = strings.ReplaceAll(address, "%ROOT_DIR%", f.root)
	return strings.ReplaceAll(address, ",", string(filepath.Separator))
}

func (f *Fileprocessor) CheckPathConsistency(path string) bool {
	// Check if path belongs to root dir
	if ok := strings.Contains(path, f.root); !ok {
		return ok
	}
	// Only absolute paths are available
	if ok := filepath.IsAbs(path); !ok {
		return ok
	}
	// Must exist and be a directory
	dir, err := os.Stat(path)
	if err != nil {
		return false
	}
	if !dir.IsDir() {
		return false
	}

	return true
}

func (f *Fileprocessor) EscapeWithUser(address string, u int) string {
	address = strings.ReplaceAll(address, fmt.Sprint(u)+string(filepath.Separator), "")
	address = strings.ReplaceAll(address, f.root, "%ROOT_DIR%")
	return strings.ReplaceAll(address, string(filepath.Separator), ",")
}

func (f *Fileprocessor) ExtractUser(address string, u uint) string {
	return strings.ReplaceAll(address, "%ROOT_DIR%,"+fmt.Sprint(u), "%ROOT_DIR%")
}

func (f *Fileprocessor) InsertUser(address string, u uint) string {
	return strings.ReplaceAll(address, "%ROOT_DIR%", "%ROOT_DIR%,"+fmt.Sprint(u))
}

func (f *Fileprocessor) SplitPath(p string) (d string, c string) {
	//Escaped path needs to be converted before and after
	if CheckEscaped(p) {
		p = f.UnEscapeAddress(p)
		d, c = filepath.Split(p)
		d = f.EscapeAddress(strings.TrimSuffix(d, string(filepath.Separator)))
	} else {
		d, c = filepath.Split(p)
		d = strings.TrimSuffix(d, string(filepath.Separator))
	}
	return d, c
}

func (p *Fileprocessor) CheckEscaped(path string) bool {
	return strings.Contains(path, ",")
}

func (p *Fileprocessor) AddEventIntoBuffer(object string, et FSEventType, ignore bool) {
	p.abMutex.Lock()
	p.actionBuffer[object] = append(p.actionBuffer[object], BufferedAction{Action: et, Ignore: ignore, Timestamp: time.Now()})
	p.abMutex.Unlock()
}
