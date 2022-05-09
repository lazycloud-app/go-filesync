package proto

import (
	"time"
)

var (
	Terminator       = byte('\n')
	TerminatorString = "\n"
)

type (
	// MessageType represents sync message types
	MessageType int

	// Message is the model for base sync message
	Message struct {
		Type      MessageType
		Token     string
		Timestamp time.Time
		Payload   []byte
	}

	Auth struct {
		Login      string
		Password   string
		DeviceName string
		Label      string
		Session    string
	}

	StartSync struct {
		Type     SyncType
		NotAfter time.Time
		Objects  []string // Only for SyncTypes that include specific files or folders
	}

	Token struct {
		Token      string
		SessionKey string
	}

	OK struct {
		OK            bool
		HumanReadable string
	}
)

const (
	messages_start MessageType = iota

	MessageError
	MessageAuth            // Request for token by login & password
	MessageToken           // Response with newly generated token for client
	MessageDirSyncReq      // Request for filelist (client -> server) with own filelist in specific dir
	MessageDirSyncResp     // Response from server with filelist (server -> client) and list of files to upload on server in specific dir
	MessageGetFile         // Request to get []bytes of specific file (client -> server)
	MessageSendFile        // Response with []bytes of specific file (client <-> server)
	MessageConnectionEnd   // Message to close connetion (client <-> server)
	MessageOK              // The other side correctly understood previous message OR not (client <-> server)
	MessageStartSync       // The other party is ready to recieve filesystem events
	MessageEndSync         // The other side doesn't need filesystem events anymore
	MessageCloseConnection // The other side doesn't need the connection anymore **POSSIBLY REDUNDANT**
	MessageSyncEvent       // Notify other perties that file or dir were created / changed / deleted
	MessageHandshake
	MessageHello
	MessageFileParts
	MessageFileEnd
	MessageWarning

	messages_end
	// For readability
	MessageUnknownType
)

func (m *Message) CheckType() bool {
	if messages_start < m.Type && m.Type < messages_end {
		return true
	}
	return false
}

func (m MessageType) String() string {
	if m < messages_start || messages_end > m {
		return "Unknown"
	}
	return [...]string{"Unknown", "Error", "Authorization", "New token", "MessageDirSyncReq", "MessageDirSyncResp", "MessageGetFile", "MessageSendFile", "MessageConnectionEnd", "OK", "MessageStartSync", "MessageEndSync", "MessageCloseConnection", "MessageSyncEvent", "MessageHandshake", "MessageHello", "MessageFileParts", "MessageFileEnd", "Warning", "Unknown", "Unknown"}[m]
}
