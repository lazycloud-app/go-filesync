package messenger

import (
	"fmt"

	"github.com/lazycloud-app/go-filesync/v1/basic/proto"
)

var (
	Version    = 1
	VerComment = "basic"
)

type (
	Messenger struct {
		ver        int
		verComment string
		token      string
		send       *proto.Message // Message to send
		recieved   *proto.Message // Message recieved
		bsent      int
		brec       int
	}
)

func New() *Messenger {
	m := new(Messenger)
	m.Init()
	return m
}

func (m *Messenger) Init() {
	m.ver = Version
	m.verComment = VerComment
	m.send = new(proto.Message)
	m.recieved = new(proto.Message)
}

func (m *Messenger) SetToken(t string) {
	m.token = t
	m.send.Token = t
}

func (m *Messenger) Version() string {
	return fmt.Sprintf("%d (%s)", m.ver, m.verComment)
}

func (m *Messenger) Recieved() *proto.Message {
	return m.recieved
}

func (m *Messenger) SentBytes() int {
	return m.bsent
}

func (m *Messenger) RecievedBytes() int {
	return m.brec
}
