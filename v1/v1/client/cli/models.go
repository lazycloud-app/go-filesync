package cli

import (
	"github.com/lazycloud-app/go-filesync/common/syncerr"
	"github.com/lazycloud-app/go-filesync/config"
	"github.com/lazycloud-app/go-fsp-proto/ver"
)

type (
	Client struct {
		ver       ver.AppVersion
		conf      config.ClientConfV1
		errorChan chan (*syncerr.IE)
	}
)
