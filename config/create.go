package config

import (
	"encoding/json"
	"os"
	"strings"
)

func Create_v1(file *os.File) error {
	var nc ClientConfV1
	nc.CONFIG_VER = 1

	m, err := json.Marshal(&nc)
	if err != nil {
		return err
	}

	str := string(m)
	str = strings.ReplaceAll(str, `{"`, "{\n	\"")
	str = strings.ReplaceAll(str, `,"`, ",\n	\"")
	str = strings.ReplaceAll(str, `"}`, "\"\n}")

	_, err = file.WriteString(str)
	if err != nil {
		return err
	}

	return nil
}
