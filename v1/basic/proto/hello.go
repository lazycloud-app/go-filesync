package proto

import "github.com/lazycloud-app/go-filesync/ver"

type (
	// Intension signals the purpose of the connection that yet to be made
	Intension int

	// Hello used typically to send message from client to server (discovering possibility of sync)
	//
	// It has all the fields that server may use to determine whether the client suits server's demands
	Hello struct {
		ConnectIntension      Intension
		PartyName             string
		AppVersion            ver.AppVersion
		OwnerContacts         string
		MaxClients            int
		MaxConnectionsPerUser int
		MaxSessionsPerUser    int // Session basically means device
		MaxFileSize           int
		MaxFiles              int
		MaxFilesPerUser       int
	}

	// Handshake signals the client to pass on authorization information and tells sever's rules
	Handshake struct {
		PartyName             string
		AppVersion            ver.AppVersion
		OwnerContacts         string
		AdditionalServerRules []string
		MaxClients            int
		MaxConnectionsPerUser int
		MaxFileSize           int
		MaxFiles              int
		MaxFilesPerUser       int
		SyncTypesPermitted    []SyncType
	}
)

const (
	intensions_start Intension = iota

	IntensionClient
	IntensionMirror

	intensions_end
	// Represents Unknown Intension, just for readability
	IntensionUnknown
)

func (i Intension) Check() bool {
	if intensions_start > i && i > intensions_end {
		return false
	}
	return true
}

func (i Intension) String() string {
	if !i.Check() {
		return "Unknown Intension"
	}
	return [...]string{"Unknown Intension", "Client", "Mirror server", "Unknown Intension", "Unknown Intension"}[i]
}
