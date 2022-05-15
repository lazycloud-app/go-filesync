package server

import (
	"crypto/tls"
	"fmt"
)

func (s *Server) Listen() {
	// Prepare TLS
	tlsConfig, err := s.GetTLSCOnfig()
	if err != nil {
		s.Fatal(fmt.Errorf("[Listen] error getting key pair -> %w", err))
	}
	l, err := tls.Listen("tcp", ":"+s.config.Port, tlsConfig)
	if err != nil {
		s.Fatal(fmt.Errorf("[Listen] error listening -> %w", err))
	}

	s.InfoCyan(fmt.Sprintf("Listening on %s:%s", s.config.HostName, s.config.Port))

	// Handle close errors
	defer func() {
		err := l.Close()
		if err != nil {
			s.Error(fmt.Errorf("[Listen] error closing listener: %w", err))
		}
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			s.Error(fmt.Errorf("(%v)[Listen] error accepting connection: %w", conn.RemoteAddr(), err))
		}

		// Accept new communication request & add to current pool
		comm, err := s.pool.AcceptCommunication(conn, s.config.ServerToken)
		if err != nil {
			s.Error(fmt.Errorf("(%v)[Listen] error starting communication: %w", conn.RemoteAddr(), err))
		}
		// Start reading & sending messages
		go s.pool.StartCommunication(comm)
	}
}

func (s *Server) GetTLSCOnfig() (tlsConfig *tls.Config, err error) {
	cert, err := tls.LoadX509KeyPair(s.config.CertPath, s.config.KeyPath)
	if err != nil {
		return tlsConfig, fmt.Errorf("[GetTLSCOnfig] error getting key pair -> %w", err)
	}
	tlsConfig = &tls.Config{Certificates: []tls.Certificate{cert}}

	return tlsConfig, nil
}
