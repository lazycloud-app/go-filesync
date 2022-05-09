package server

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/gofrs/uuid"
	"github.com/lazycloud-app/go-filesync/users"
	"github.com/lazycloud-app/go-filesync/v1/basic/cloud/events"
	"github.com/lazycloud-app/go-filesync/v1/basic/proto"
)

func (s *Server) StartSync(req *proto.Message, client *Connection) (cResp ParseError) {
	if client.Token != req.Token {
		cResp.Err = true
		cResp.Text = "Wrong security token"
		cResp.Type = proto.ErrAccessDenied
		s.Send(EventType("warn"), events.SourceSyncServerListener.String(), fmt.Errorf("(%v) [StartSync] wrong token", client.ip))
		client.ClientErrors++
		return
	}
	// Starting sync channel with the other party
	client.StateChan <- SyncStart

	return
}

func (s *Server) ParseHello(req *proto.Message, client *Connection) (hello proto.Hello, response ParseError) {
	err := json.Unmarshal(req.Payload, &hello)
	if err != nil {
		response.Err = true
		response.Text = "Can not read the message"
		response.Type = proto.ErrBrokenMessage
		s.Send(EventType("error"), events.SourceSyncServerListener.String(), fmt.Errorf("(%v) [ParseHello] error Unmarshal: %w", client.ip, err))
		client.ClientErrors++
		return
	}
	// Checking demands for clients
	// Just major versions
	if hello.AppVersion.Major != s.appVersion.Major {
		response.Err = true
		response.Text = fmt.Sprintf("Incompatible app versions: the server is at '%s'", s.appVersion.String())
		response.Type = proto.ErrIncompatibleAppVersion
		return
	}
	// And full protocol comparsion
	if !s.appVersion.Proto.FullComparsion(hello.AppVersion.Proto) {
		response.Err = true
		response.Text = fmt.Sprintf("Incompatible protocol: the server uses '%s'", s.appVersion.Proto.String())
		response.Type = proto.ErrIncompatibleProtocol
		return
	}
	if !hello.ConnectIntension.Check() {
		response.Err = true
		response.Text = "Unknown sync intension"
		response.Type = proto.ErrIntensionUnknown
		return
	}
	if hello.ConnectIntension == proto.IntensionMirror {
		response.Err = true
		response.Text = "Mirroring not allowed"
		response.Type = proto.ErrIntensionRejected
		return
	}

	//send ErrIncompatibleConditions in case max file size or other cient conditions are not OK

	return
}

func (s *Server) ParseAuth(req *proto.Message, client *Connection) (token string, response ParseError) {
	var auth proto.Auth
	err := json.Unmarshal(req.Payload, &auth)
	if err != nil {
		response.Err = true
		response.Text = "Can not read the message"
		response.Type = proto.ErrBrokenMessage
		s.Send(EventType("error"), events.SourceSyncServerListener.String(), fmt.Errorf("(%v) [ParseAuth] error Unmarshal: %w", client.ip, err))
		client.ClientErrors++
		return
	}
	if auth.Login == "" || auth.Password == "" {
		response.Err = true
		response.Text = "Empty login or password"
		response.Type = proto.ErrBrokenMessage
		return
	}
	// Check credentials
	ok, uid, rIP, err := users.ValidateCreds(auth.Login, auth.Password, s.db)
	if err != nil {
		s.Send(EventType("error"), events.SourceSyncServerListener.String(), fmt.Errorf("(%v) [ValidateCreds] error validating creds: %w", client.ip, err))
		client.ServerErrors++
		response.Err = true
		response.Text = "Unknown error"
		response.Type = proto.ErrInternal
		return
	}
	if !ok {
		response.WrongCreds()
		return
	}
	client.Uid = uid
	// Token for the connection
	token, err = users.GenerateToken()
	if err != nil {
		if err != nil {
			s.Send(EventType("error"), events.SourceSyncServerListener.String(), fmt.Errorf("(%v) [GenerateToken] error generating: %w", client.ip, err))
			client.ServerErrors++
			response.Err = true
			response.Text = "Unknown error"
			response.Type = proto.ErrInternal
			return
		}
	}
	client.Token = token
	// Generate session hash if the client is not in session hall
	if auth.Session == "" {
		session, err := uuid.NewV4()
		if err != nil {
			s.Send(EventType("error"), events.SourceSyncServerListener.String(), fmt.Errorf("(%v) [uuid.NewV4] error making uuid: %w", client.ip, err))
			client.ServerErrors++
			response.Err = true
			response.Text = "Unknown error"
			response.Type = proto.ErrInternal
			return
		}
		// If user is allowed to use one IP only and connection ip is wrong
		cIP := client.ip.String()
		if rIP != "" && cIP != rIP {
			response.Err = true
			response.Text = fmt.Sprintf("Connection from ip %s is not permitted", cIP)
			response.Type = proto.ErrAccessDenied
			return
		}
		client.Session = session.String()
		s.AddSession(client.Session, uid, client.ip)
	} else {
		client.Session = auth.Session
		// Checking here if this session exists could be necessary in case MaxClientSessions is not limited
		// In a perfect world we don't need to, but there is a vulnerability:
		// If somebody evil enough creates a client-app that will not store session key
		// And instead will create connections with empty key (e.g. gets new key for every file downloaded)
		// Then potentially it will make allocated server memory increase dramatically
		// Because of creating a huge session keys storage
		// (I never checked, but better to avoid potential risks)
		// So just use the MaxClientSessions var or check here if session exists
	}
	return
}

func (s *Server) AddSession(key string, uid uint, ip net.Addr) {
	se := Session{
		Uid: uid,
		Key: key,
		IP:  ip,
	}
	s.sessionsPoolMutex.Lock()
	s.activeSessions[uid] = append(s.activeSessions[uid], &se)
	s.activeSessionsNumber++
	s.sessionsPoolMutex.Unlock()
}

func (s *Server) AddConnectionToPool(c *Connection) {
	s.connectionPoolMutex.Lock()
	s.activeConnections = append(s.activeConnections, c)
	s.activeConnectionsNumber++
	s.connectionPoolMutex.Unlock()
}

func (s *Server) ActiveConnectionsLess() {
	s.connectionPoolMutex.Lock()
	s.activeConnectionsNumber--
	s.connectionPoolMutex.Unlock()
}
