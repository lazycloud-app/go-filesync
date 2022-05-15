package messenger

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/lazycloud-app/go-filesync/v1/basic/proto"
	"github.com/lazycloud-app/go-fsp-proto/ver"
)

func (m *Messenger) SendError(t proto.ErrorType, exp string, c interface{}) error {
	if ok := t.CheckErrorType(); !ok {
		return fmt.Errorf("[SendError] incorrect err type")
	}
	errorPayload := proto.Error{
		Type:      t,
		Explained: exp,
	}

	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(errorPayload)
	if err != nil {
		return fmt.Errorf("[SendError] Encode: %w", err)
	}

	m.send.Type = proto.MessageError
	m.send.Payload = b.Bytes()
	err = m.Push(c)
	if err != nil {
		return err
	}

	return nil
}

func (m *Messenger) Handshake(c interface{}, partyName string, appVersion ver.AppVersion, ownerContacts string, maxClients int, maxConnectionsPerUser int, maxFileSize int) error {
	payload := proto.Handshake{
		PartyName:             partyName,
		AppVersion:            appVersion,
		OwnerContacts:         ownerContacts,
		MaxClients:            maxClients,
		MaxConnectionsPerUser: maxConnectionsPerUser,
		MaxFileSize:           maxFileSize,
	}

	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(payload)
	if err != nil {
		return err
	}

	m.send.Payload = b.Bytes()
	m.send.Type = proto.MessageHandshake

	err = m.Push(c)
	if err != nil {
		return err
	}
	return err
}

func (m *Messenger) SendSyncEvent(c interface{}, e proto.FSEvent) error {
	m.send.Type = proto.MessageSyncEvent

	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(e)
	if err != nil {
		return err
	}

	m.send.Payload = b.Bytes()
	m.send.Timestamp = time.Now()

	err = m.Push(c)
	if err != nil {
		return err
	}

	return nil
}

func (m *Messenger) Push(c interface{}) error {
	var bs int
	response, err := json.Marshal(m.send)
	if err != nil {
		return err
	}
	m.send.Timestamp = time.Now()

	nc, ok := c.(*net.Conn)
	if ok {
		bs, err = io.WriteString(*nc, string(response)+proto.TerminatorString)
		m.bsent += bs
		if err != nil {
			return fmt.Errorf("[Push] (net.Conn) error writing response -> %v", err)
		}
		return err
	}

	tc, ok := c.(*tls.Conn)
	if ok {
		bs, err = io.WriteString(tc, string(response)+proto.TerminatorString)
		m.bsent += bs
		if err != nil {
			return fmt.Errorf("[Push] (tls.Conn) error writing response -> %v", err)
		}
		return err
	}

	if !ok {
		return fmt.Errorf("[Push] c is not a suitable connection")
	}

	return nil
}

// Send concates all message parts and calls Push to send payload to other party
func (m *Messenger) SendMessage(payload interface{}, t proto.MessageType, c interface{}) error {
	// Using type assertion here (by payload, for example) would just make code less readable
	// So I think it's easier to just define type when calling to Send
	m.send.Type = t
	if !m.send.CheckType() {
		return fmt.Errorf("[Send] unknown message type")
	}

	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(payload)
	if err != nil {
		return fmt.Errorf("[Send] error encoding -> %w", err)
	}

	m.send.Payload = b.Bytes()
	err = m.Push(c)
	if err != nil {
		return fmt.Errorf("[Send] error pushing -> %w", err)
	}

	return nil
}

func (m *Messenger) SendFileData(c interface{}, f *proto.SyncFileData) error {
	m.send.Type = proto.MessageSendFile
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(f)
	if err != nil {
		return err
	}

	m.send.Payload = b.Bytes()
	err = m.Push(c)
	if err != nil {
		return fmt.Errorf("[SendFileData] error pushing -> %w", err)
	}
	return nil
}

func (m *Messenger) SendFilePart(c interface{}, p []byte) error {
	m.send.Type = proto.MessageFileParts

	m.send.Payload = p
	err := m.Push(c)
	if err != nil {
		return fmt.Errorf("[SendFilePart] error pushing -> %w", err)
	}
	return nil
}

func (m *Messenger) SendFileEnd(c interface{}) error {
	m.send.Type = proto.MessageFileEnd

	m.send.Payload = []byte{}
	err := m.Push(c)
	if err != nil {
		return fmt.Errorf("[SendFilePart] error pushing -> %w", err)
	}
	return nil
}
