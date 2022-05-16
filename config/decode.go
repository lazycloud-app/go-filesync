package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

func Decode_v1(f string) (*ClientConfV1, error) {
	var c ClientConfV1
	b, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, err
	}
	//Replacing possible bad bytes to avoid json errors
	//Not all users can understand errors like "invalid character 'C' in string escape code"
	//So it's just good for usability
	for n, v := range b {
		if v == '\\' {
			b[n] = '/'
		}
	}
	err = json.Unmarshal(b, &c)
	if err != nil {
		return nil, err
	}
	//Avoiding possible bad user input in config file / CLI or cases when user saved config from other filesystem
	c.EscapeBadFilepaths()

	if c.CONFIG_VER != 1 {
		return nil, fmt.Errorf("config version mismatch (want 1, got %d", c.CONFIG_VER)
	}

	return &c, nil
}
