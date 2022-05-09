package client

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gofrs/uuid"
)

func (c *Client) Sync() {
	conn, err := c.InitTLSConnection()
	if err != nil {
		c.Logger.FatalBackRed(fmt.Errorf("[Sync] error -> %w", err))
	}
	c.Logger.InfoCyan(fmt.Sprintf("Connected to %s:%d", c.Config.ServerAddress, c.Config.ServerPort))
	// Sending Hello and discovering if server suits client requirements
	err = c.Hello(conn)
	if err != nil {
		c.Logger.Error("[Sync] error -> ", err)
		return
	}
	fmt.Println("HELLO")

	err = c.Auth(conn)
	if err != nil {
		c.Logger.Error("[Sync] error -> ", err)
		return
	}
	fmt.Println("AUTH")

	fmt.Println("START SYNC")
	err = c.SyncStart(conn)
	if err != nil {
		c.Logger.Error("[Sync] error -> ", err)
		return
	}

	fmt.Println("SYNC ENDED")

}

func (c *Client) AddActionToBuffer(object string, eType fsnotify.Op) {
	c.ActionsBufferMutex.Lock()
	c.ActionsBuffer[object] = append(c.ActionsBuffer[object],
		BufferedAction{
			Action:    eType,
			Timestamp: time.Now(),
		})
	c.ActionsBufferMutex.Unlock()
}

func (c *Client) CreateFileInCache() (cacheAddress string, err error) {
	u, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	tempPath := c.Config.CacheDir + string(filepath.Separator) + "cache_getfile" + u.String()
	if err := os.MkdirAll(c.Config.CacheDir, os.ModePerm); err != nil {
		return "", err
	}
	theFile, err := os.Create(tempPath)
	if err != nil {
		return "", err
	}
	theFile.Close()

	return tempPath, nil
}
