package messenger

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/lazybark/go-helpers/hasher"
	"github.com/lazycloud-app/go-filesync/v1/basic/fsworker"
	"github.com/lazycloud-app/go-filesync/v1/basic/imp"
	"github.com/lazycloud-app/go-filesync/v1/basic/proto"
	"github.com/lazycloud-app/go-filesync/ver"
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

func (m *Messenger) SendSyncEvent(c interface{}, eventType fsnotify.Op, o interface{}) error {
	m.send.Type = proto.MessageSyncEvent
	var payload proto.SyncEvent

	file, okf := o.(proto.File)
	if okf {
		payload = proto.SyncEvent{
			Type:         imp.SyncEventFromWatcherEvent(eventType),
			ObjectType:   proto.ObjectFile,
			Name:         file.Name,
			Path:         file.Path,
			Hash:         file.Hash,
			NewUpdatedAt: file.FSUpdatedAt,
		}
	}

	folder, okd := o.(proto.Folder)
	if okd {
		payload = proto.SyncEvent{
			Type:         imp.SyncEventFromWatcherEvent(eventType),
			ObjectType:   proto.ObjectDir,
			Name:         folder.Name,
			Path:         folder.Path,
			Hash:         "",
			NewUpdatedAt: folder.FSUpdatedAt,
		}
	}

	if !okf && !okd {
		return errors.New("[SendSyncEvent] o is not suitable object")
	}

	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(payload)
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
func (m *Messenger) Send(payload interface{}, t proto.MessageType, c interface{}) error {
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

func (m *Messenger) SendFile(c interface{}, file string, fw *fsworker.Fsworker, owner uint) (bytesSent int, err error) {
	m.send.Type = proto.MessageSendFile
	var payload proto.SyncFileData

	stat, err := os.Stat(file)
	if err != nil {
		return
	}
	fmt.Println("DDD1")

	fileData, err := os.Open(file)
	if err != nil {
		return 0, fmt.Errorf("can not open file -> %w", err)
	}
	defer fileData.Close()
	fmt.Println("DDD2")

	dir, _ := filepath.Split(file)
	dir = strings.TrimSuffix(dir, string(filepath.Separator))

	hash, err := hasher.HashFilePath(file, hasher.SHA256, 8192)
	if err != nil {
		return
	}
	fmt.Println("DDD3")

	payload = proto.SyncFileData{
		Name: filepath.Base(file),
		// Extracting user is necessary: client does not know anything about filestructure on the server and slient's uid
		Path:        fw.ExtractUser(fw.EscapeAddress(dir), owner),
		Hash:        hash,
		Size:        stat.Size(),
		FSUpdatedAt: stat.ModTime(),
		Type:        filepath.Ext(file),
	}

	// Sending info about file data
	b := new(bytes.Buffer)
	err = json.NewEncoder(b).Encode(payload)
	if err != nil {
		return
	}

	m.send.Payload = b.Bytes()

	err = m.Push(c)
	if err != nil {
		return
	}
	fmt.Println("DDD4")

	//data := []byte{}

	// TLS record size can be up to 16KB but some extra bytes may apply
	// Read this before you change
	// https://hpbn.co/transport-layer-security-tls/#optimize-tls-record-size
	buf := make([]byte, 15360)
	n := 0

	r := bufio.NewReader(fileData)

	m.send.Type = proto.MessageFileParts

	for {
		n, err = r.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("ERRRE", err)
			return
		}

		m.send.Payload = buf[:n]

		err = m.Push(c)
		if err != nil {
			return
		}

	}
	fmt.Println("DDD5")

	m.send.Type = proto.MessageFileEnd

	err = m.Push(c)
	if err != nil {
		return
	}
	fmt.Println("DDD6")

	return
}
