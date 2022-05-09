package client

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"

	"github.com/lazycloud-app/go-filesync/v1/basic/proto"
)

type (
	ParseError struct {
		Err  bool
		Type proto.ErrorType
		Text error
	}
)

func (e ParseError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Type.String(), e.Text)
}

// ReadStream reads tls stream until MessageTerminator occurs.
//
// It is equal to bufio.NewReader().ReadBytes()
// except context. ReadStream will break in case ctx.Done() occurs.
// Useful to make reading timeouts and prevent routine leak or client freezing.
func (c *Client) ReadStream(ctx context.Context, conn *tls.Conn) ([]byte, error) {
	netData := bufio.NewReader(conn)

	var ret []byte
	read := 0
	b := make([]byte, 128)
	// Read until MessageTerminator or error
	for {
		select {
		case <-ctx.Done():
			// Break by timeout
			return nil, fmt.Errorf("[ReadStream] reading limits reached")
		default:
			n, err := netData.Read(b)
			if err != nil {
				return nil, fmt.Errorf("[ReadStream] reading error: %w", err)
			}
			read += n
			for num, by := range b[:n] {
				if by == proto.Terminator {
					ret = append(ret, b[:num]...)
					return ret, nil
				}
			}
			if c.Config.MaxMessageSize > 0 && c.Config.MaxMessageSize >= read {
				return nil, fmt.Errorf("[ReadStream] message size limits reached")
			}
			ret = append(ret, b[:n]...)

			if err == io.EOF {
				return nil, fmt.Errorf("[ReadStream] stream closed")
			}
		}
	}
}

func (c *Client) ProcessErrorPayload(payload []byte) error {
	var e proto.Error
	err := json.Unmarshal(payload, &e)
	if err != nil {
		return fmt.Errorf("[ProcessErrorPayload] error unmarshalling -> %w", err)
	}

	return fmt.Errorf("server responded with an error: %s (%s)", e.Type, e.Explained)
}
