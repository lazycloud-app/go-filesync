package config

import (
	"path/filepath"
	"strings"
)

type (
	ClientConfV1 struct {
		CONFIG_VER            int
		LOGIN                 string
		PASSWORD              string
		SERVER_CERT_FILE      string
		SERVER_ADDRESS        string
		SERVER_PORT           int
		DIR_LOGS              string
		DIR_CACHE             string
		FILE_SYSTEM_ROOT_PATH string
		DB_FILE_NAME          string
	}
)

//EscapeBadFilepaths replaces possibly wrong bits in filepaths in case config file has unpredictable separators.
//Useful to keep single-slashed Windows filepaths in json-encoded config
func (c *ClientConfV1) EscapeBadFilepaths() {
	c.FILE_SYSTEM_ROOT_PATH = strings.ReplaceAll(c.FILE_SYSTEM_ROOT_PATH, "/", string(filepath.Separator))
	c.FILE_SYSTEM_ROOT_PATH = strings.ReplaceAll(c.FILE_SYSTEM_ROOT_PATH, `\`, string(filepath.Separator))
	c.FILE_SYSTEM_ROOT_PATH = strings.ReplaceAll(c.FILE_SYSTEM_ROOT_PATH, string(filepath.Separator)+string(filepath.Separator), string(filepath.Separator))

	c.SERVER_CERT_FILE = strings.ReplaceAll(c.SERVER_CERT_FILE, "/", string(filepath.Separator))
	c.SERVER_CERT_FILE = strings.ReplaceAll(c.SERVER_CERT_FILE, `\`, string(filepath.Separator))
	c.SERVER_CERT_FILE = strings.ReplaceAll(c.SERVER_CERT_FILE, string(filepath.Separator)+string(filepath.Separator), string(filepath.Separator))

	c.SERVER_ADDRESS = strings.ReplaceAll(c.SERVER_ADDRESS, "/", string(filepath.Separator))
	c.SERVER_ADDRESS = strings.ReplaceAll(c.SERVER_ADDRESS, `\`, string(filepath.Separator))
	c.SERVER_ADDRESS = strings.ReplaceAll(c.SERVER_ADDRESS, string(filepath.Separator)+string(filepath.Separator), string(filepath.Separator))

	c.DIR_LOGS = strings.ReplaceAll(c.DIR_LOGS, "/", string(filepath.Separator))
	c.DIR_LOGS = strings.ReplaceAll(c.DIR_LOGS, `\`, string(filepath.Separator))
	c.DIR_LOGS = strings.ReplaceAll(c.DIR_LOGS, string(filepath.Separator)+string(filepath.Separator), string(filepath.Separator))

	c.DIR_CACHE = strings.ReplaceAll(c.DIR_CACHE, "/", string(filepath.Separator))
	c.DIR_CACHE = strings.ReplaceAll(c.DIR_CACHE, `\`, string(filepath.Separator))
	c.DIR_CACHE = strings.ReplaceAll(c.DIR_CACHE, string(filepath.Separator)+string(filepath.Separator), string(filepath.Separator))

	c.FILE_SYSTEM_ROOT_PATH = strings.ReplaceAll(c.FILE_SYSTEM_ROOT_PATH, "/", string(filepath.Separator))
	c.FILE_SYSTEM_ROOT_PATH = strings.ReplaceAll(c.FILE_SYSTEM_ROOT_PATH, `\`, string(filepath.Separator))
	c.FILE_SYSTEM_ROOT_PATH = strings.ReplaceAll(c.FILE_SYSTEM_ROOT_PATH, string(filepath.Separator)+string(filepath.Separator), string(filepath.Separator))

	c.DB_FILE_NAME = strings.ReplaceAll(c.DB_FILE_NAME, "/", string(filepath.Separator))
	c.DB_FILE_NAME = strings.ReplaceAll(c.DB_FILE_NAME, `\`, string(filepath.Separator))
	c.DB_FILE_NAME = strings.ReplaceAll(c.DB_FILE_NAME, string(filepath.Separator)+string(filepath.Separator), string(filepath.Separator))

}
