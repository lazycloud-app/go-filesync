package fs

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func CheckEscaped(path string) bool {
	return strings.Contains(path, ",")
}

func GetOwner(root string, path string) uint {
	// ownerId will extract user id that owns directory in root
	var ownerId = regexp.MustCompile(`%ROOT_DIR%,(\d*)`)
	// Only work with escaped addresses to avoid ambiguity
	if !CheckEscaped(path) {
		path = EscapeAddress(root, path)
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

func EscapeAddress(escape string, path string) string {
	address := strings.ReplaceAll(path, escape, "%ROOT_DIR%")
	return strings.ReplaceAll(address, string(filepath.Separator), ",")
}

func UnEscapeAddress(escape string, path string) string {
	address := strings.ReplaceAll(path, "%ROOT_DIR%", escape)
	return strings.ReplaceAll(address, ",", string(filepath.Separator))
}

func InsertUser(address string, u uint) string {
	return strings.ReplaceAll(address, "%ROOT_DIR%", "%ROOT_DIR%,"+fmt.Sprint(u))
}

func ExtractUser(address string, u uint) string {
	return strings.ReplaceAll(address, "%ROOT_DIR%,"+fmt.Sprint(u), "%ROOT_DIR%")
}

func SplitPath(p string) (string, string) {
	d, c := filepath.Split(p)
	d = strings.TrimSuffix(d, string(filepath.Separator))
	return d, c
}
