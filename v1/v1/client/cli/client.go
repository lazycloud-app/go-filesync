package cli

import (
	"fmt"

	"github.com/lazycloud-app/go-filesync/common/syncerr"
)

//New creates new client with last available version and loads config using CLI
func New(conf string) *Client {
	c := new(Client)
	c.ver = version
	c.errorChan = make(chan *syncerr.IE)
	//Getting config
	if conf == "" {
		conf = "conf"
	}
	c.ConfigOpenCLI(conf)

	return c
}

//Start runs client
func (c *Client) Start() {
	//Error processing routine goes first
	go c.ErrorsRoutine()
}

//ErrorsRoutine processes errors and warnings & logs events into command line
func (c *Client) ErrorsRoutine() {
	for err := range c.errorChan {
		if err.Log {
			fmt.Println(err.Err)
		}
	}
}
