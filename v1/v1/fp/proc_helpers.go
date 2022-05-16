package fp

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func (fp *FP) EscapeAddress(address string) string {
	address = strings.ReplaceAll(address, fp.root, RootPointer)
	return strings.ReplaceAll(address, string(filepath.Separator), Delim)
}

func (fp *FP) UnEscapeAddress(address string) string {
	address = strings.ReplaceAll(address, RootPointer, fp.root)
	return strings.ReplaceAll(address, Delim, string(filepath.Separator))
}

func (fp *FP) CheckPathConsistency(path string) bool {
	// Check if path belongs to root dir
	if ok := strings.Contains(path, fp.root); !ok {
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

func (fp *FP) SplitPath(p string) (d string, c string) {
	//Escaped path needs to be converted before and after
	if CheckEscaped(p) {
		p = fp.UnEscapeAddress(p)
		d, c = filepath.Split(p)
		d = fp.EscapeAddress(strings.TrimSuffix(d, string(filepath.Separator)))
	} else {
		d, c = filepath.Split(p)
		d = strings.TrimSuffix(d, string(filepath.Separator))
	}
	return d, c
}

func (fp *FP) GetOwner(path string) uint {
	// ownerId will extract user id that owns directory in root
	var ownerId = regexp.MustCompile(fmt.Sprintf("%s%s(\\d*)", RootPointer, Delim))
	// Only work with escaped addresses to avoid ambiguity
	if !CheckEscaped(path) {
		path = fp.EscapeAddress(path)
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
