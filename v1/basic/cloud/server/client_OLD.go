package server

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/lazycloud-app/go-filesync/v1/basic/messenger"
)

type (
	ConnEvent int

	Connection struct {
		Active         bool // false means that server will delete all connection data on next cleaner iteration
		ClosedByServer bool
		SyncActive     bool // Client <-> Server filesync is active
		EventsChan     chan (FSEventNotification)
		ip             net.Addr
		DeviceName     string
		PartyName      string
		BytesSent      int
		BytesRecieved  int
		ClientErrors   uint
		ServerErrors   uint
		Uid            uint
		Token          string
		ConnectAt      time.Time
		DisconnectedAt time.Time
		StateChan      chan (ConnEvent) // Channel for closing the sync routine
		Session        string           // Sessions exist to concat separate connections data for statistic
	}

	Client struct {
	}
)

const (
	ConnClose ConnEvent = iota + 1
	SyncStart
	SyncStop
)

func NewConnection(ip net.Addr, active bool) *Connection {
	c := new(Connection)
	c.EventsChan = make(chan FSEventNotification)
	c.StateChan = make(chan ConnEvent)
	c.Active = active
	c.ip = ip
	c.ConnectAt = time.Now()

	return c
}

func (c *Connection) Close() {
	c.Active = false
	c.SyncActive = false
	c.DisconnectedAt = time.Now()

	c.StateChan <- ConnClose
}

// Routine to maintain current connection
func (c *Connection) Maintain(token string, net net.Conn) {
	m := messenger.New()
	m.SetToken(token)
	for {
		select {
		case data, ok := <-c.EventsChan:
			if !ok {
				io.WriteString(net, "Channel closed")
				// Close sync in pool
				c.SyncActive = false
				return
			}
			fmt.Println("Sending to client")
			err := m.SendSyncEvent(net, data.Event.Op, data.Object)
			if err != nil {
				fmt.Println(err)
			}
			c.BytesSent = m.SentBytes()

		case e, ok := <-c.StateChan:
			if !ok || e == ConnClose {
				io.WriteString(net, "Connection closed")

				fmt.Println("Connection closing")
				// Close sync in pool
				return
			} else if e == SyncStart {
				c.SyncActive = true
				//	c.Send(EventType("info"), events.SourceSyncServerListener.String(), fmt.Sprintf("(%v) sync started", net.RemoteAddr()))

			} else if e == SyncStop {
				c.SyncActive = false
				//	c.Send(EventType("info"), events.SourceSyncServerListener.String(), fmt.Sprintf("(%v) sync stopped", net.RemoteAddr()))
			}
		}
	}
}
