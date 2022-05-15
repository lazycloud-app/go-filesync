package proto

type (
	// ErrorType represents sync error code and human-readable name
	ErrorType int
	// Error is the model for error message payload
	Error struct {
		Type      ErrorType
		Explained string
	}
	// ErrorClient represents errors that occur during parsing/sending sync messages that come from client
	ErrorClient struct {
		Err   error
		Descr string
	}
	// ErrorServer represents errors that occur during parsing/sending sync messages that come from server
	ErrorServer struct {
		Err   error
		Descr string
	}
)

// Error codes
const (
	errors_start ErrorType = iota

	ErrBrokenMessage
	ErrUnknownMessageType
	ErrAccessDenied
	ErrInternal
	ErrTooMuchServerErrors
	ErrTooMuchClientErrors
	ErrTooMuchClients
	ErrTooMuchConnections
	ErrIncompatibleAppVersion
	ErrIncompatibleProtocol
	ErrIncompatibleConditions
	ErrHaveNewerVersion

	ErrIntensionUnknown
	ErrIntensionRejected

	errors_end
)

// String() returns human-readable name of error code
func (e ErrorType) String() string {
	if !e.CheckErrorType() {
		return "Unknown err type"
	}
	return [...]string{
		"Unknown err type",
		"Broken message",
		"Unknown message type",
		"Access denied",
		"Sync app internal error",
		"Too much errors on server side",
		"Too much errors on client side",
		"Server has reached client limits",
		"Client has reached connections limit",
		"Incompatible app version",
		"Incompatible protocol version",
		"Incompatible conditions",
		"I have newer version",
		"Unknown connection intension",
		"Unpermitted connection intension",
		"Unknown err type"}[e]
}

// SyncBreaker determines whether the error type is deadly for sync.
// Usually it's the ones that client can not recover from
//
// If error theoretically can be recovered from - better to return false,
// so client will decide individually
func (e ErrorType) SyncBreaking() bool {
	if e == ErrAccessDenied || e == ErrTooMuchServerErrors || e == ErrTooMuchClientErrors || e == ErrTooMuchClients || e == ErrIncompatibleAppVersion || e == ErrIncompatibleConditions {
		return true
	}
	return false
}

// CheckErrorType() checks error code for consistency
func (e *ErrorType) CheckErrorType() bool {
	if errors_start > *e && *e > errors_end {
		return false
	}
	return true
}

func (e ErrorServer) Error() string {
	return e.Err.Error()
}

func (e ErrorClient) Error() string {
	return e.Err.Error()
}
