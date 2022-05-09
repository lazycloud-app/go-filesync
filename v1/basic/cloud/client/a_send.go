package client

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/lazycloud-app/go-filesync/v1/basic/imp"
	"github.com/lazycloud-app/go-filesync/v1/basic/messenger"
	"github.com/lazycloud-app/go-filesync/v1/basic/proto"
)

func (c *Client) Hello(conn *tls.Conn) error {
	// We send Hello message
	m := messenger.New()
	rec := m.Recieved()
	c.Logger.Info("Sending Hello")

	payload := proto.Hello{
		ConnectIntension:      proto.IntensionClient,
		PartyName:             "test_client_1",
		AppVersion:            c.AppVersion,
		OwnerContacts:         "lazybark.dev@gmail.com",
		MaxClients:            0,
		MaxConnectionsPerUser: 15,
		MaxFileSize:           2048,
	}

	err := m.Send(payload, proto.MessageHello, conn)
	if err != nil {
		return fmt.Errorf("[Hello] error sending Hello: %w", err)
	}
	c.Stat.BytesSent += m.SentBytes()

	// Wait for answer
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		netData, err := c.ReadStream(ctx, conn)
		cancel()
		if err != nil {
			return fmt.Errorf("[Hello] error reading response from %v:%v -> %w", c.Config.ServerAddress, c.Config.ServerPort, err)
		}

		//var m messenger.Messenger
		err = m.ParseRecieved(&netData)
		if err != nil {
			return fmt.Errorf("[Hello] broken message from %v:%v -> %w", c.Config.ServerAddress, c.Config.ServerPort, err)
		}
		c.Stat.BytesRecieved += m.RecievedBytes()

		// The only suitable state is getting Handshake here
		// Any other message type = error
		if rec.Type == proto.MessageHandshake {
			var handshake proto.Handshake
			err = json.Unmarshal(rec.Payload, &handshake)
			if err != nil {
				return fmt.Errorf("[Hello] error unmarshalling Handshake -> %w", err)
			}
			c.Logger.InfoGreen(fmt.Sprintf("Got Handshake from %v. Server: %s (v %s). Max file size %d, max connections per user %d", conn.RemoteAddr(), handshake.PartyName, handshake.AppVersion, handshake.MaxFileSize, handshake.MaxConnectionsPerUser))
			c.Logger.InfoGreen(fmt.Sprintf("Owner info: %s", handshake.OwnerContacts))

			return nil
		} else if rec.Type == proto.MessageError {
			return fmt.Errorf("[Hello] %w", c.ProcessErrorPayload(rec.Payload))
		} else {
			return fmt.Errorf("[Hello] server responded with and unexpected message type %s", rec.Type)
		}

	}
}

func (c *Client) Auth(conn *tls.Conn) error {
	m := messenger.New()
	rec := m.Recieved()
	c.Logger.Info("Sending Auth")

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "NO NAME"
	}

	payload := proto.Auth{
		Login:      c.Config.Login,
		Password:   c.Config.Password,
		DeviceName: hostname,
		Label:      "NO LABEL",
		Session:    c.SessionKey,
	}

	err = m.Send(payload, proto.MessageAuth, conn)
	if err != nil {
		return fmt.Errorf("[Auth] error sending Auth: %w", err)
	}
	c.Stat.BytesSent += m.SentBytes()

	// Wait for answer
	retry := 0
	for {
		if retry >= 5 {
			// We break in case server spams with unknown messages
			return fmt.Errorf("[Auth] no answer from server")
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		netData, err := c.ReadStream(ctx, conn)
		cancel()
		if err != nil {
			return fmt.Errorf("[Auth] error reading response from %v:%v -> %w", c.Config.ServerAddress, c.Config.ServerPort, err)
		}

		err = m.ParseRecieved(&netData)
		if err != nil {
			return fmt.Errorf("[Auth] broken message from %v:%v -> %w", c.Config.ServerAddress, c.Config.ServerPort, err)
		}
		c.Stat.BytesRecieved += m.RecievedBytes()

		// The only suitable state is getting Token here
		// Any other message type = error
		if rec.Type == proto.MessageToken {
			var token proto.Token
			err = json.Unmarshal(rec.Payload, &token)
			if err != nil {
				return fmt.Errorf("[Auth] error unmarshalling Token -> %w", err)
			}
			c.CurrentToken = token.Token
			if c.SessionKey == "" {
				c.SessionKey = token.SessionKey
			}
			if !c.SyncActive {
				c.SyncActive = true
				c.FileGetter = make(chan proto.GetFile)
			}
			// Start routine that will make download requests
			go c.RequestRoutine()
			return nil
		} else if rec.Type == proto.MessageError {
			return fmt.Errorf("[Auth] %w", c.ProcessErrorPayload(rec.Payload))
		}
		retry++
	}
}

// SyncStart negins endless fs sync process, possible exit only through an error or SyncEnd()
func (c *Client) SyncStart(conn *tls.Conn) error {
	m := messenger.New()
	m.SetToken(c.CurrentToken)
	rec := m.Recieved()

	payload := proto.StartSync{
		Type:     proto.SyncTypeFullSignal,
		NotAfter: time.Now().Add(25 * time.Hour),
	}

	err := m.Send(payload, proto.MessageStartSync, conn)
	if err != nil {
		return fmt.Errorf("[SyncStart] error sending Start: %w", err)
	}
	c.Stat.BytesSent += m.SentBytes()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	netData, err := c.ReadStream(ctx, conn)
	cancel()
	if err != nil {
		return fmt.Errorf("[SyncStart] error reading response from %v:%v -> %w", c.Config.ServerAddress, c.Config.ServerPort, err)
	}

	err = m.ParseRecieved(&netData)
	if err != nil {
		return fmt.Errorf("[SyncStart] broken message from %v:%v -> %w", c.Config.ServerAddress, c.Config.ServerPort, err)
	}
	c.Stat.BytesRecieved += m.RecievedBytes()

	if rec.Type == proto.MessageOK {
		fmt.Println("SYNC OK")
	} else if rec.Type == proto.MessageError {
		return fmt.Errorf("[SyncStart] %w", c.ProcessErrorPayload(rec.Payload))
	} else {
		return fmt.Errorf("[SyncStart] server responded with and unexpected message type %s", rec.Type)
	}

	// Endless sync events await
	// No timeouts this time
	for {
		netData, err := bufio.NewReader(conn).ReadBytes(proto.Terminator)
		if err != nil {
			if errors.As(err, &io.ErrClosedPipe) {
				return fmt.Errorf("server has closed the connection")
			}
			return fmt.Errorf("ERROR DUING SYNC, %w", err)
		}

		err = m.ParseRecieved(&netData)
		if err != nil {
			return fmt.Errorf("[SyncStart] broken message from %v:%v -> %w", c.Config.ServerAddress, c.Config.ServerPort, err)
		}

		if rec.Type == proto.MessageSyncEvent {
			event, err := imp.ParseSyncEvent(rec.Payload)
			if err != nil {
				c.Logger.Error(fmt.Errorf("(%v)[ParseSyncEvent] broken payload: %w", conn.RemoteAddr(), err))
				continue
			}
			fmt.Println("SYNC EVENT", event.Type.String())

			// Process removing
			if event.Type == proto.ObjectRemoved {
				c.AddActionToBuffer(c.FW.UnEscapeAddress(filepath.Join(event.Path, event.Name)), fsnotify.Remove)

				sResp := c.ProcessObjectRemoved(event)
				if sResp.Err {
					c.Logger.Error("[Active Sync] error -> ", err)
					// Server doesn't need to know error description in case error is internal
					if sResp.Type != proto.ErrInternal {
						err = m.SendError(sResp.Type, sResp.Text.Error(), &conn)
						if err != nil {
							c.Logger.Error(fmt.Errorf("[Active Sync] error sending err: %s", err))
							continue
						}
					} else {
						err = m.SendError(sResp.Type, "", &conn)
						if err != nil {
							c.Logger.Error(fmt.Errorf("[Active Sync] error sending err: %s", err))
							continue
						}
					}
					c.Stat.BytesSent += m.SentBytes()
				}
				continue
			} else if event.Type == proto.ObjectCreated {
				sResp := c.ProcessObjectCreated(event)
				if sResp.Err {
					c.Logger.Error("[Active Sync] error -> ", err)
					// Server doesn't need to know error description in case error is internal
					if sResp.Type != proto.ErrInternal {
						err = m.SendError(sResp.Type, sResp.Text.Error(), &conn)
						if err != nil {
							c.Logger.Error(fmt.Errorf("[Active Sync] error sending err: %s", err))
							continue
						}
					} else {
						err = m.SendError(sResp.Type, "", &conn)
						if err != nil {
							c.Logger.Error(fmt.Errorf("[Active Sync] error sending err: %s", err))
							continue
						}
					}
					c.Stat.BytesSent += m.SentBytes()
				}
				continue
			} else if event.Type == proto.ObjectUpdated {
				sResp := c.ProcessObjectUpdated(event)
				if sResp.Err {
					c.Logger.Error("[Active Sync] error -> ", err)
					// Server doesn't need to know error description in case error is internal
					if sResp.Type != proto.ErrInternal {
						err = m.SendError(sResp.Type, sResp.Text.Error(), &conn)
						if err != nil {
							c.Logger.Error(fmt.Errorf("[Active Sync] error sending err: %s", err))
							continue
						}
					} else {
						err = m.SendError(sResp.Type, "", &conn)
						if err != nil {
							c.Logger.Error(fmt.Errorf("[Active Sync] error sending err: %s", err))
							continue
						}
					}
					c.Stat.BytesSent += m.SentBytes()
				}
				continue
			} else {
				c.Logger.Error("[Active Sync] server sent unexpected file action type: %s", event.Type)
				continue
			}
		} else if rec.Type == proto.MessageError {
			//PARSE error AND print
			//if it's deadly - exit
			c.Logger.Error("[Active Sync] server sent error")
			continue
		} else if rec.Type == proto.MessageWarning {
			//PARSE WARNING AND print
			c.Logger.Error("[Active Sync] server sent warning")
			continue
		} else {
			c.Logger.Error(fmt.Sprintf("[Active Sync] server sent unexpected message type: %s", rec.Type))
			continue
		}
	}

}

func (c *Client) SyncEnd(conn *tls.Conn) error {

	return nil
}

func (c *Client) RequestRoutine() {
	fmt.Println("RequestRoutine started")
	for fileToGet := range c.FileGetter {
		fmt.Println("Getting", fileToGet.Name)

		c.FilesInRow = append(c.FilesInRow, fileToGet)
		go c.GetFile(&fileToGet)
	}
	fmt.Println("RequestRoutine stopped")
}
