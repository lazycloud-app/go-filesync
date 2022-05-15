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

	err = json.Unmarshal(b, &c)
	if err != nil {
		return nil, err
	}

	if c.CONFIG_VER != 1 {
		return nil, fmt.Errorf("config version mismatch (want 1, got %d", c.CONFIG_VER)
	}

	return &c, nil
}
