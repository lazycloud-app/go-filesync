package imp

type (
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

func (e ErrorServer) Error() string {
	return e.Err.Error()
}

func (e ErrorClient) Error() string {
	return e.Err.Error()
}
