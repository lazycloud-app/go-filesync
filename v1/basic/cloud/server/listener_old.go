package server

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"path/filepath"

	"github.com/lazycloud-app/go-filesync/v1/basic/cloud/events"
	"github.com/lazycloud-app/go-filesync/v1/basic/messenger"
	"github.com/lazycloud-app/go-filesync/v1/basic/proto"
)

func (s *Server) Listen() {
	// Prepare TLS
	tlsConfig, err := s.GetTLSCOnfig()
	if err != nil {
		s.Send(EventType("fatal"), events.SourceSyncServer.String(), fmt.Errorf("[Listen] error getting key pair -> %w", err))
	}
	l, err := tls.Listen("tcp", ":"+s.config.Port, tlsConfig)
	if err != nil {
		s.Send(EventType("fatal"), events.SourceSyncServer.String(), fmt.Errorf("[Listen] error listening -> %w", err))
	}
	s.Send(EventType("cyan"), events.SourceSyncServer.String(), fmt.Sprintf("Listening on %s:%s", s.config.HostName, s.config.Port))

	// Handle close errors
	defer func() {
		err := l.Close()
		if err != nil {
			s.Send(EventType("error"), events.SourceSyncServerListener.String(), fmt.Errorf("error closing listener -> %w", err))
		}
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			s.Send(EventType("error"), events.SourceSyncServerListener.String(), fmt.Errorf("(%v)[Listen] error accepting connection: %w", conn.RemoteAddr(), err))
		}
		go s.AcceptConnection(conn)
	}
}

func (s *Server) AcceptConnection(conn net.Conn) {
	// client represents new client from the server's perspective
	client := NewConnection(conn.RemoteAddr(), true)
	s.Send(EventType("info"), events.SourceSyncServerListener.String(), fmt.Sprintf("(%v)[New Connection] Active: %d", client.ip, s.activeConnectionsNumber+1))
	// maintain connections pool to control sync stages
	s.AddConnectionToPool(client)
	// messnger manages requests & responses
	m := messenger.New()
	m.SetToken(s.config.ServerToken)
	rec := m.Recieved()

	for {
		fmt.Printf("BYTES SENT %d, RECIEVED %d \n", client.BytesRecieved, client.BytesSent)
		// Read tls stream until next message separator
		streamBytes, err := bufio.NewReader(conn).ReadBytes(proto.Terminator)
		if err != nil {
			// If connection closed - break the cycle
			if errors.As(err, &io.ErrClosedPipe) {
				client.Close()
				s.ActiveConnectionsLess()
				if client.ClosedByServer {
					s.Send(EventType("info"), events.SourceSyncServerListener.String(), fmt.Sprintf("(%v)[Connection closed by server] - recieved %d bytes, sent %d bytes. Errors: %d. Active connections: %d", client.ip, client.BytesRecieved, client.BytesSent, client.ClientErrors+client.ServerErrors, s.activeConnectionsNumber))
				} else {
					s.Send(EventType("info"), events.SourceSyncServerListener.String(), fmt.Sprintf("(%v)[Connection closed by other party] - recieved %d bytes, sent %d bytes. Errors: %d. Active connections: %d", client.ip, client.BytesRecieved, client.BytesSent, client.ClientErrors+client.ServerErrors, s.activeConnectionsNumber))
				}
				break
			}
			client.ClientErrors++
			continue
		}

		err = m.ParseRecieved(&streamBytes)
		if err != nil {
			s.Send(EventType("error"), events.SourceSyncServerListener.String(), fmt.Errorf("(%v)[message.Parse] - broken message: %v", conn.RemoteAddr(), err))
			client.ClientErrors++
			continue
		}
		client.BytesRecieved = m.RecievedBytes()

		// Now: try to parse payload depending on req.Type field
		// Type assertions here would just make code more complicated
		// So looking for req.Type is more simple solution
		var respondError *ParseError
		if rec.Type == proto.MessageHello {
			fmt.Println("HELLO")
			hello, cResp := s.ParseHello(rec, client)
			// Non-empty response means error for client
			if cResp.Err {
				cResp.Stage = "Handshake"
				respondError = &cResp
			} else {
				client.PartyName = hello.PartyName

				err := m.Handshake(conn, s.config.ServerName, s.appVersion, s.config.OwnerContacts, 0, 15, 2048)
				if err != nil {
					s.Send(EventType("error"), events.SourceSyncServerListener.String(), fmt.Errorf("(%v) error making response 'Handshake': %w", conn.RemoteAddr(), err))
					client.ServerErrors++
					continue
				}

				// Routine to maintain current connection
				go client.Maintain(s.config.ServerToken, conn)
				client.BytesSent = m.SentBytes()
				fmt.Println("HELLO")
				continue
			}

		} else if rec.Type == proto.MessageAuth {
			fmt.Println("AUTH")
			token, cResp := s.ParseAuth(rec, client)
			// Non-empty response means error for client
			if cResp.Err {
				cResp.Stage = "Token"
				respondError = &cResp
			} else {
				// Now sending connection token
				err := m.Send(proto.Token{Token: token, SessionKey: client.Session}, proto.MessageToken, &conn)
				if err != nil {
					s.Send(EventType("error"), events.SourceSyncServerListener.String(), fmt.Errorf("(%v) [Send] error making response 'Token': %w", conn.RemoteAddr(), err))
					client.ServerErrors++
					continue
				}
				client.BytesSent = m.SentBytes()
				fmt.Println("AUTH")
				continue
			}
		} else if rec.Type == proto.MessageOK {
			/*ok, err := req.ValidateOK()
			if err != nil {
				s.evProc.Send(EventType("error"), events.SourceSyncServerListener.String(), fmt.Sprintf("(%v) error parsing OK: %s", conn.RemoteAddr(), err))

				_, err := resp.ReturnError(&conn, proto.ErrBrokenMessage)
				if err != nil {
					s.evProc.Send(EventType("error"), events.SourceSyncServerListener.String(), fmt.Errorf("(%v) error making response: %w", conn.RemoteAddr(), err))
				}
				if client.ClientErrors >= s.config.MaxClientErrors {
					client.Close()
					s.evProc.Send(EventType("info"), events.SourceSyncServerListener.String(), fmt.Sprintf("(%v)[Connection closed] - recieved %d bytes, sent %d bytes. Errors: %d. Active connections: %d", client.ip, client.BytesRecieved, client.BytesSent, client.ClientErrors+client.ServerErrors, s.activeConnectionsNumber))
				}
				continue

			}
			if !ok.OK {
				s.evProc.Send(EventType("warn"), events.SourceSyncServerListener.String(), fmt.Sprintf("(%v) client sent error: %s", conn.RemoteAddr(), ok.HumanReadable))
				continue
			}
			continue*/

		} else if rec.Type == proto.MessageStartSync {
			fmt.Println("START SYNC")
			cResp := s.StartSync(rec, client)
			// Non-empty response means error for client
			if cResp.Err {
				cResp.Stage = "START SYNC"
				respondError = &cResp
			} else {
				// Now sending OK
				err := m.Send(&proto.OK{OK: true, HumanReadable: "Starting sync of type ..."}, proto.MessageOK, &conn)
				if err != nil {
					s.Send(EventType("error"), events.SourceSyncServerListener.String(), fmt.Errorf("(%v) [Send] error making response 'OK': %w", conn.RemoteAddr(), err))
					client.ServerErrors++
				}

				fmt.Println("START SYNC")
				client.BytesSent = m.SentBytes()
				continue
			}

		} else if rec.Type == proto.MessageEndSync {
			/*
				ok := client.Token == rec.Token

				if !ok {
					s.evProc.Send(EventType("yellow"), events.SourceSyncServerListener.String(), fmt.Sprintf("(%v) wrong token", conn.RemoteAddr()))

					err := m.SendError(proto.ErrAccessDenied, "Wrong token", &conn)
					if err != nil {
						s.evProc.Send(EventType("error"), events.SourceSyncServerListener.String(), fmt.Errorf("(%v) error making response: %w", conn.RemoteAddr(), err))
						if client.ServerErrors >= s.config.MaxServerErrors && s.config.MaxServerErrors > 0 {
							s.evProc.Send(EventType("warn"), events.SourceSyncServerListener.String(), fmt.Sprintf("(%v)[config.MaxServerErrors] - 'client error per connection' limit reached, conn will be closed", conn.RemoteAddr()))
							client.Close()
							s.evProc.Send(EventType("info"), events.SourceSyncServerListener.String(), fmt.Sprintf("(%v)[Connection closed] - recieved %d bytes, sent %d bytes. Errors: %d. Active connections: %d", client.ip, client.BytesRecieved, client.BytesSent, client.ClientErrors+client.ServerErrors, s.activeConnectionsNumber))
							return
						}
						continue
					}

					continue
				}

				// Starting sync channel with the other party
				client.StateChan <- SyncStop
				// Requesting to start syncing back
				resp.ReturnInfoMessage(&conn, proto.MessageOK)
				resp.ReturnInfoMessage(&conn, proto.MessageEndSync)
				continue*/
		} else if rec.Type == proto.MessageCloseConnection {
			/*
				ok := client.Token == rec.Token

				if !ok {
					s.evProc.Send(EventType("yellow"), events.SourceSyncServerListener.String(), fmt.Sprintf("(%v) wrong token", conn.RemoteAddr()))

					bytesSent, err := resp.ReturnError(&conn, proto.ErrAccessDenied)
					if err != nil {
						s.evProc.Send(EventType("error"), events.SourceSyncServerListener.String(), fmt.Errorf("(%v) error making response: %w", conn.RemoteAddr(), err))
						if client.ServerErrors >= s.config.MaxServerErrors && s.config.MaxServerErrors > 0 {
							s.evProc.Send(EventType("info"), events.SourceSyncServerListener.String(), fmt.Sprintf("(%v)[config.MaxServerErrors] - 'client error per connection' limit reached, conn will be closed", conn.RemoteAddr()))
							client.Close()
							s.evProc.Send(EventType("info"), events.SourceSyncServerListener.String(), fmt.Sprintf("(%v)[Connection closed] - recieved %d bytes, sent %d bytes. Errors: %d. Active connections: %d", client.ip, client.BytesRecieved, client.BytesSent, client.ClientErrors+client.ServerErrors, s.activeConnectionsNumber))
							return
						}
						continue
					}
					if s.config.CountStats {
						fmt.Println(bytesSent)
					}
					if s.config.ServerVerboseLogging && !s.config.SilentMode {
						s.evProc.Send(EventType("info"), events.SourceSyncServerListener.String(), fmt.Sprintf("%v - recieved %d bytes, sent %d bytes", conn.RemoteAddr(), len(netData), bytesSent))
					}
					continue
				}

				// Requesting to start syncing back
				resp.ReturnInfoMessage(&conn, proto.MessageOK)
				client.Close()
				s.evProc.Send(EventType("info"), events.SourceSyncServerListener.String(), fmt.Sprintf("(%v)[Connection closed] - recieved %d bytes, sent %d bytes. Errors: %d. Active connections: %d", client.ip, client.BytesRecieved, client.BytesSent, client.ClientErrors+client.ServerErrors, s.activeConnectionsNumber))
				return*/
		} else if rec.Type == proto.MessageGetFile {
			fmt.Println("GET FILE")
			getFile, err := m.ParseGetFile()
			if err != nil {
				fmt.Println(err)
			}

			go func() {
				_, err := m.SendFile(conn, s.fw.UnEscapeAddress(s.fw.InsertUser(filepath.Join(getFile.Path, getFile.Name), client.Uid)), s.fw, client.Uid)
				if err != nil {
					fmt.Println(err)
				}
				client.BytesSent = m.SentBytes()
				fmt.Println("SENT FILE")
			}()

			fmt.Println("GET FILE")
			continue
		} else {
			fmt.Println("UNKNOWN MESSAGE TYPE")
			respondError.Err = true
			respondError.Stage = "Unknown message"
			respondError.Type = proto.ErrUnknownMessageType
			respondError.Text = ""
			client.ClientErrors++
		}

		if respondError.Err {
			err := m.SendError(respondError.Type, respondError.Text, &conn)
			if err != nil {
				s.Send(EventType("error"), events.SourceSyncServerListener.String(), fmt.Errorf("(%v) error sending err ('%s'): %w", client.ip, respondError.Stage, err))
				client.ServerErrors++
			}
			client.BytesSent = m.SentBytes()
			continue
		}
	}
	client.Close()
	s.Send(EventType("info"), events.SourceSyncServerListener.String(), fmt.Sprintf("(%v)[Connection closed] - recieved %d bytes, sent %d bytes. Errors: %d. Active connections: %d", client.ip, client.BytesRecieved, client.BytesSent, client.ClientErrors+client.ServerErrors, s.activeConnectionsNumber))

}

func (s *Server) GetTLSCOnfig() (tlsConfig *tls.Config, err error) {
	cert, err := tls.LoadX509KeyPair(s.config.CertPath, s.config.KeyPath)
	if err != nil {
		return tlsConfig, fmt.Errorf("[GetTLSCOnfig] error getting key pair -> %w", err)
	}
	tlsConfig = &tls.Config{Certificates: []tls.Certificate{cert}}

	return tlsConfig, nil
}
