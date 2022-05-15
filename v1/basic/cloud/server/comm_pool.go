package server

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/lazybark/go-helpers/hasher"
	"github.com/lazybark/go-pretty-code/logs"
	"github.com/lazycloud-app/go-filesync/users"
	"github.com/lazycloud-app/go-filesync/v1/basic/fs"
	"github.com/lazycloud-app/go-filesync/v1/basic/imp"
	"github.com/lazycloud-app/go-filesync/v1/basic/messenger"
	"github.com/lazycloud-app/go-filesync/v1/basic/proto"
	"gorm.io/gorm"
)

type (
	Comm struct {
		n              string
		active         bool
		closedByServer bool
		syncActive     bool
		connectAt      time.Time
		disconnectedAt time.Time
		token          string
		uid            uint
		login          string
		deviceName     string
		partyName      string
		sess           *Session
		conn           *net.Conn
		eventsChan     chan (proto.FSEvent)
		stateChan      chan (ConnEvent)

		*messenger.Messenger
	}

	CommPool struct {
		pool   map[string]*Comm
		m      *sync.RWMutex
		server ServerData
		db     *gorm.DB
		fp     *fs.Fileprocessor

		logs.Logger
	}

	Session struct {
		sessionKey string
		restrictIP bool
		ip         net.Addr
	}

	ConnEvent int
)

const (
	ConnClose ConnEvent = iota + 1
	SyncStart
	SyncStop
)

func (c *CommPool) AcceptCommunication(conn net.Conn, token string) (*Comm, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	m := messenger.New()

	nc := Comm{
		id.String(),
		true,
		false,
		false,
		time.Now(),
		time.Time{},
		"",
		0,
		"",
		"",
		"",
		&Session{ip: conn.RemoteAddr()},
		&conn,
		make(chan proto.FSEvent),
		make(chan ConnEvent),

		m,
	}
	nc.SetToken(token)
	c.AddCommunicationToPool(&nc, id.String())
	c.Info(fmt.Sprintf("New Connection from %v", nc.sess.ip))

	return &nc, nil

}

func (c *CommPool) AddCommunicationToPool(comm *Comm, k string) {
	c.m.Lock()
	c.pool[k] = comm
	c.m.Unlock()
}

func (co *Comm) CommClose() {
	co.active = false
	co.syncActive = false
	co.disconnectedAt = time.Now()

	co.stateChan <- ConnClose
}

func (co *Comm) ClosedByServer() bool {
	return co.closedByServer
}

func (s *Session) IP() net.Addr {
	return s.ip
}

func (co *Comm) IP() net.Addr {
	return co.sess.IP()
}

func (co *Comm) PartyName(pn string) {
	co.partyName = pn
}

func (co *Comm) SetLogin(l string) {
	co.login = l
}

func (co *Comm) Login() string {
	return co.login
}

func (co *Comm) SetClientToken(t string) {
	co.token = t
}

func (co *Comm) SetUID(i uint) {
	co.uid = i
}

func (co *Comm) Maintain() {
	for {
		select {
		case data, ok := <-co.eventsChan:
			if !ok {
				io.WriteString(*co.conn, "Channel closed")
				// Close sync in pool
				co.syncActive = true
				return
			}
			fmt.Println("Sending to client")
			err := co.SendSyncEvent(*co.conn, data)
			if err != nil {
				fmt.Println(err)
			}

		case e, ok := <-co.stateChan:
			if !ok || e == ConnClose {
				io.WriteString(*co.conn, "Connection closed")

				fmt.Println("Connection closing")
				// Close sync in pool
				return
			} else if e == SyncStart {
				co.syncActive = true
				fmt.Println("Sync started")
				//	c.Send(EventType("info"), events.SourceSyncServerListener.String(), fmt.Sprintf("(%v) sync started", net.RemoteAddr()))

			} else if e == SyncStop {
				co.syncActive = false
				//	c.Send(EventType("info"), events.SourceSyncServerListener.String(), fmt.Sprintf("(%v) sync stopped", net.RemoteAddr()))
			}
		}
	}
}

func (c *CommPool) StartCommunication(comm *Comm) {
	rec := comm.Recieved()

	for {
		fmt.Printf("BYTES SENT %d, RECIEVED %d \n", comm.RecBytes(), comm.SBytes())
		// Read tls stream until next message separator
		streamBytes, err := bufio.NewReader(*comm.conn).ReadBytes(proto.Terminator)
		if err != nil {
			// If connection closed - break the cycle
			if errors.As(err, &io.ErrClosedPipe) {
				comm.CommClose()
				if comm.ClosedByServer() {
					c.Info(fmt.Sprintf("(%v)[Connection closed by server] - recieved %d bytes, sent %d bytes. Errors: %d", comm.IP(), comm.RecBytes(), comm.SBytes(), comm.ClientErrors()+comm.ServerErrors()))
				} else {
					c.Info(fmt.Sprintf("(%v)[Connection closed by other party] - recieved %d bytes, sent %d bytes. Errors: %d", comm.IP(), comm.RecBytes(), comm.SBytes(), comm.ClientErrors()+comm.ServerErrors()))
				}
				break
			}
			comm.AddClientErrors(1)
			continue
		}

		err = comm.ParseRecieved(&streamBytes)
		if err != nil {
			c.Error(fmt.Errorf("(%v)[message.Parse] - broken message: %v", comm.IP(), err))
			comm.AddClientErrors(1)
			continue
		}

		// Now: try to parse payload depending on req.Type field
		// Type assertions here would just make code more complicated
		// So looking for req.Type is more simple solution
		if rec.Type == proto.MessageHello {
			fmt.Println("HELLO")
			var hello proto.Hello
			err := json.Unmarshal(rec.Payload, &hello)
			if err != nil {
				c.Error(fmt.Errorf("(%v) [ParseHello] error Unmarshal: %w", comm.IP(), err))
				err := comm.SendError(proto.ErrBrokenMessage, "Can not read the message", comm.conn)
				if err != nil {
					c.Error(fmt.Errorf("(%v) error sending err ('Handshake'): %w", comm.IP(), err))
					comm.AddClientErrors(1)
				}
				continue
			}
			// Checking demands for clients
			// Just major versions
			if hello.AppVersion.Major != c.server.appVersion.Major {
				err := comm.SendError(proto.ErrIncompatibleAppVersion, fmt.Sprintf("Incompatible app versions: the server is at '%s'", c.server.appVersion.String()), comm.conn)
				if err != nil {
					c.Error(fmt.Errorf("(%v) error sending err ('Handshake'): %w", comm.IP(), err))
					comm.AddClientErrors(1)
				}
				continue
			}
			// And full protocol comparsion
			if !c.server.appVersion.Proto.FullCompareTo(hello.AppVersion.Proto) {
				err := comm.SendError(proto.ErrIncompatibleProtocol, fmt.Sprintf("Incompatible protocol: the server uses '%s'", c.server.appVersion.Proto.String()), comm.conn)
				if err != nil {
					c.Error(fmt.Errorf("(%v) error sending err ('Handshake'): %w", comm.IP(), err))
					comm.AddClientErrors(1)
				}
				continue
			}
			if !hello.ConnectIntension.Check() {
				err := comm.SendError(proto.ErrIntensionUnknown, "Unknown sync intension", comm.conn)
				if err != nil {
					c.Error(fmt.Errorf("(%v) error sending err ('Handshake'): %w", comm.IP(), err))
					comm.AddClientErrors(1)
				}
				continue
			}
			if hello.ConnectIntension == proto.IntensionMirror {
				err := comm.SendError(proto.ErrIntensionRejected, "Mirroring not allowed", comm.conn)
				if err != nil {
					c.Error(fmt.Errorf("(%v) error sending err ('Handshake'): %w", comm.IP(), err))
					comm.AddClientErrors(1)
				}
				continue
			}
			//send ErrIncompatibleConditions in case max file size or other cient conditions are not OK

			comm.PartyName(hello.PartyName)
			err = comm.Handshake(comm.conn, c.server.name, c.server.appVersion, c.server.ownerContacts, 0, 15, 2048)
			if err != nil {
				c.Error(fmt.Errorf("(%v) error making response 'Handshake': %w", comm.IP(), err))
				comm.AddServerErrors(1)
				continue
			}
			// Routine to maintain current connection
			go comm.Maintain()
			fmt.Println("HELLO")
			continue

		} else if rec.Type == proto.MessageAuth {
			fmt.Println("AUTH")
			var auth proto.Auth
			err := json.Unmarshal(rec.Payload, &auth)
			if err != nil {
				comm.AddClientErrors(1)
				err := comm.SendError(proto.ErrBrokenMessage, "Can not read the message", comm.conn)
				if err != nil {
					c.Error(fmt.Errorf("(%v) error sending err ('Auth'): %w", comm.IP(), err))
					comm.AddClientErrors(1)
				}
				continue
			}
			if auth.Login == "" || auth.Password == "" {
				comm.AddClientErrors(1)
				err := comm.SendError(proto.ErrBrokenMessage, "Empty login or password", comm.conn)
				if err != nil {
					c.Error(fmt.Errorf("(%v) error sending err ('Auth'): %w", comm.IP(), err))
					comm.AddClientErrors(1)
				}
				continue
			}
			// Check credentials
			ok, uid, rIP, err := users.ValidateCreds(auth.Login, auth.Password, c.db)
			if err != nil {
				c.Error(fmt.Errorf("(%v) [ValidateCreds] error validating creds: %w", comm.IP(), err))
				comm.AddClientErrors(1)
				err := comm.SendError(proto.ErrInternal, "Unknown error", comm.conn)
				if err != nil {
					c.Error(fmt.Errorf("(%v) error sending err ('Auth'): %w", comm.IP(), err))
					comm.AddServerErrors(1)
				}
				continue
			}
			if !ok {
				c.Error(fmt.Errorf("(%v) [ValidateCreds] error validating creds: %w", comm.IP(), err))
				err := comm.SendError(proto.ErrAccessDenied, "Wrong login or password", comm.conn)
				if err != nil {
					c.Error(fmt.Errorf("(%v) error sending err ('Auth'): %w", comm.IP(), err))
					comm.AddServerErrors(1)
				}
				continue
			}
			comm.SetUID(uid)
			comm.SetLogin(auth.Login)
			// Token for the connection
			token, err := users.GenerateToken()
			if err != nil {
				if err != nil {
					c.Error(fmt.Errorf("(%s) [GenerateToken] error generating: %w", comm.Login(), err))
					comm.AddServerErrors(1)
					err := comm.SendError(proto.ErrInternal, "Unknown error", comm.conn)
					if err != nil {
						c.Error(fmt.Errorf("(%v) error sending err ('Auth'): %w", comm.IP(), err))
						comm.AddServerErrors(1)
					}
					continue
				}
			}
			comm.SetClientToken(token)
			// Generate session hash if the client is not in session hall
			if auth.SessionKey == "" {
				session, err := uuid.NewV4()
				if err != nil {
					c.Error(fmt.Errorf("(%v) [uuid.NewV4] error making uuid: %w", comm.IP(), err))
					comm.AddServerErrors(1)
					err := comm.SendError(proto.ErrInternal, "Unknown error", comm.conn)
					if err != nil {
						c.Error(fmt.Errorf("(%v) error sending err ('Auth'): %w", comm.IP(), err))
						comm.AddServerErrors(1)
					}
					continue
				}
				// If user is allowed to use one IP only and connection ip is wrong
				cIP := comm.IP().String()
				if rIP != "" && cIP != rIP {
					err := comm.SendError(proto.ErrAccessDenied, fmt.Sprintf("Connection from ip %s is not permitted", cIP), comm.conn)
					if err != nil {
						c.Error(fmt.Errorf("(%v) error sending err ('Auth'): %w", comm.IP(), err))
						comm.AddServerErrors(1)
					}
					continue
				}
				comm.sess.sessionKey = session.String()

			} else {
				comm.sess.sessionKey = auth.SessionKey
				// Checking here if this session exists could be necessary in case MaxClientSessions is not limited
				// In a perfect world we don't need to, but there is a vulnerability:
				// If somebody evil enough creates a client-app that will not store session key
				// And instead will create connections with empty key (e.g. gets new key for every file downloaded)
				// Then potentially it will make allocated server memory increase dramatically
				// Because of creating a huge session keys storage
				// (I never checked, but better to avoid potential risks)
				// So just use the MaxClientSessions var or check here if session exists
			}

			// Now sending connection token
			err = comm.SendMessage(proto.Token{Token: token, SessionKey: comm.sess.sessionKey}, proto.MessageToken, comm.conn)
			if err != nil {
				c.Error(fmt.Errorf("(%v) [Send] error making response 'Token': %w", comm.IP(), err))
				comm.AddServerErrors(1)
				continue
			}
			fmt.Println("AUTH")
			continue

		} else if rec.Type == proto.MessageStartSync {
			fmt.Println("START SYNC")
			if comm.token != rec.Token {
				c.Warn(fmt.Errorf("(%v) [StartSync] wrong token", comm.IP()))
				err := comm.SendError(proto.ErrAccessDenied, "Wrong security token", comm.conn)
				if err != nil {
					c.Error(fmt.Errorf("(%v) error sending err ('Auth'): %w", comm.IP(), err))
					comm.AddServerErrors(1)
				}
				comm.AddClientErrors(1)
				continue
			}
			err := comm.SendMessage(&proto.OK{OK: true, HumanReadable: "Starting sync of type ..."}, proto.MessageOK, comm.conn)
			if err != nil {
				c.Error(fmt.Errorf("(%v) [Send] error making response 'OK': %w", comm.IP(), err))
				comm.AddServerErrors(1)
			}
			// Starting sync channel with the other party
			comm.stateChan <- SyncStart
			fmt.Println("START SYNC")
		} else if rec.Type == proto.MessageEndSync {

		} else if rec.Type == proto.MessageCloseConnection {

		} else if rec.Type == proto.MessageSyncEvent {
			event, err := imp.ParseSyncEvent(rec.Payload)
			if err != nil {
				c.Logger.Error(fmt.Errorf("[ParseSyncEvent] broken payload: %w", err))
				continue
			}

			event.Object.FullPath = c.fp.InsertUser(event.Object.FullPath, comm.uid)

			fmt.Println("SYNC EVENT", event.Action.String())

			if fs.EventFromProto(event.Action) == fs.FS_UNKNOWN_ACTION {
				c.Logger.Error("[Active Sync] client sent unexpected file action type: %s", event.Action)
				continue
			} else if fs.EventFromProto(event.Action) == fs.FS_ANY_ACTION {
				c.Logger.Error("[Active Sync] client wants to process 'ANY' action whic is not supported")
				continue
			} else if fs.EventFromProto(event.Action) == fs.FS_NO_ACTION {
				continue
			}

			getFile, err := c.fp.FSEventProcessIncoming(event)
			if err != nil {
				c.Logger.Error(fmt.Errorf("[Active Sync] error in FSEventProcessIncoming: %w", err))
				continue
			}
			if getFile.Name != "" {
				fmt.Println("get file", getFile)
			}

			fmt.Println("SYNC EVENT")
		} else if rec.Type == proto.MessageGetFile {
			fmt.Println("GET FILE")

			getFile, err := comm.ParseGetFile()
			if err != nil {
				c.Error(fmt.Errorf("(%v) [GetFile] error parsing: %w", comm.IP(), err))
				err := comm.SendError(proto.ErrBrokenMessage, "Can not parse file request", comm.conn)
				if err != nil {
					c.Error(fmt.Errorf("(%v) error sending err ('Auth'): %w", comm.IP(), err))
					comm.AddClientErrors(1)
				}
				continue
			}

			fp := fs.UnEscapeAddress(c.server.rootDir, fs.InsertUser(filepath.Join(getFile.Path, getFile.Name), comm.uid))

			stat, err := os.Stat(fp)
			if err != nil {
				c.Error(fmt.Errorf("(%v) [GetFile] error getting file stat: %w", comm.IP(), err))
				err := comm.SendError(proto.ErrInternal, "Unknown error", comm.conn)
				if err != nil {
					c.Error(fmt.Errorf("(%v) error sending err ('Get file'): %w", comm.IP(), err))
					comm.AddServerErrors(1)
				}
				continue
			}

			fileData, err := os.Open(fs.UnEscapeAddress(c.server.rootDir, fs.InsertUser(fp, comm.uid)))
			if err != nil {
				c.Error(fmt.Errorf("(%v) [GetFile] error opening file: %w", comm.IP(), err))
				err := comm.SendError(proto.ErrInternal, "Unknown error", comm.conn)
				if err != nil {
					c.Error(fmt.Errorf("(%v) error sending err ('Get file'): %w", comm.IP(), err))
					comm.AddServerErrors(1)
				}
				continue
			}
			defer fileData.Close()

			hash, err := hasher.HashFilePath(fp, hasher.SHA256, 8192)
			if err != nil {
				c.Error(fmt.Errorf("(%v) [GetFile] error hashing file: %w", comm.IP(), err))
				err := comm.SendError(proto.ErrInternal, "Unknown error", comm.conn)
				if err != nil {
					c.Error(fmt.Errorf("(%v) error sending err ('Get file'): %w", comm.IP(), err))
					comm.AddServerErrors(1)
				}
				continue
			}

			dir, _ := filepath.Split(fp)
			dir = strings.TrimSuffix(dir, string(filepath.Separator))

			payload := proto.SyncFileData{
				Name: filepath.Base(fp),
				// Extracting user is necessary: client does not know anything about filestructure on the server and slient's uid
				Path:        fs.ExtractUser(fs.EscapeAddress(c.server.rootDir, dir), comm.uid),
				Hash:        hash,
				Size:        stat.Size(),
				FSUpdatedAt: stat.ModTime(),
				Type:        filepath.Ext(fp),
			}

			err = comm.SendFileData(comm.conn, &payload)
			if err != nil {
				c.Error(fmt.Errorf("(%v) [GetFile] error sending file data: %w", comm.IP(), err))
				err := comm.SendError(proto.ErrInternal, "Unknown error", comm.conn)
				if err != nil {
					c.Error(fmt.Errorf("(%v) error sending err ('Get file'): %w", comm.IP(), err))
					comm.AddServerErrors(1)
				}
				continue
			}

			// TLS record size can be up to 16KB but some extra bytes may apply
			// Read this before you change
			// https://hpbn.co/transport-layer-security-tls/#optimize-tls-record-size
			buf := make([]byte, 15360)
			n := 0

			r := bufio.NewReader(fileData)

			for {
				n, err = r.Read(buf)
				if err == io.EOF {
					break
				}
				if err != nil {
					c.Error(fmt.Errorf("(%v) [GetFile] error reading file part: %w", comm.IP(), err))
					err := comm.SendError(proto.ErrInternal, "Unknown error", comm.conn)
					if err != nil {
						c.Error(fmt.Errorf("(%v) error sending err ('Get file'): %w", comm.IP(), err))
						comm.AddServerErrors(1)
					}
					break
				}

				err = comm.SendFilePart(comm.conn, buf[:n])
				if err != nil {
					c.Error(fmt.Errorf("(%v) [GetFile] error sending file part: %w", comm.IP(), err))
					err := comm.SendError(proto.ErrInternal, "Unknown error", comm.conn)
					if err != nil {
						c.Error(fmt.Errorf("(%v) error sending err ('Get file'): %w", comm.IP(), err))
						comm.AddServerErrors(1)
					}
					break
				}
			}

			err = comm.SendFileEnd(comm.conn)
			if err != nil {
				c.Error(fmt.Errorf("(%v) [GetFile] error sending file end: %w", comm.IP(), err))
				err := comm.SendError(proto.ErrInternal, "Unknown error", comm.conn)
				if err != nil {
					c.Error(fmt.Errorf("(%v) error sending err ('Get file'): %w", comm.IP(), err))
					comm.AddServerErrors(1)
				}
			}

			fmt.Println("SENT FILE")

			fmt.Println("GET FILE")
			continue
		} else {
			fmt.Println("UNKNOWN MESSAGE TYPE")
			comm.Messenger.Err.Err = true
			comm.Messenger.Err.Stage = "Unknown message"
			comm.Messenger.Err.Type = proto.ErrUnknownMessageType
			comm.Messenger.Err.Text = ""
			comm.AddClientErrors(1)
		}
	}
	comm.CommClose()
	c.Info(fmt.Sprintf("(%v)[Connection closed] - recieved %d bytes, sent %d bytes. Errors: %d", comm.IP(), comm.RecBytes(), comm.SBytes(), comm.ClientErrors()+comm.ServerErrors()))
}
