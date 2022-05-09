package messenger

import (
	"encoding/json"
	"fmt"

	"github.com/lazycloud-app/go-filesync/v1/basic/imp"
	"github.com/lazycloud-app/go-filesync/v1/basic/proto"
)

func (m *Messenger) ParseRecieved(bytes *[]byte) error {
	m.brec += len(*bytes)
	err := json.Unmarshal(*bytes, m.recieved)
	if err != nil {
		return err
	}
	if ok := m.recieved.CheckType(); !ok {
		return imp.ErrorClient{
			Err: fmt.Errorf("[Parse] unknown message type"),
		}
	}
	return nil
}

func (m *Messenger) ParseFileData() (data proto.SyncFileData, err error) {
	err = json.Unmarshal(m.recieved.Payload, &data)
	if err != nil {
		return data, fmt.Errorf("[ParseFileData] error unmarshalling -> %w", err)
	}

	if data.Name == "" {
		return data, fmt.Errorf("[ParseFileData] broken message - file has no name")
	}
	return
}

func (m *Messenger) ParseGetFile() (getFile proto.GetFile, err error) {
	err = json.Unmarshal(m.recieved.Payload, &getFile)
	if err != nil {
		return getFile, fmt.Errorf("[ParseGetFile] error unmarshalling -> %w", err)
	}

	if getFile.Name == "" {
		err = fmt.Errorf("no name")
		return
	}
	return
}
