package server

import "github.com/lazycloud-app/go-filesync/v1/basic/proto"

type (
	ParseError struct {
		Err   bool
		Type  proto.ErrorType
		Text  string
		Stage string
	}
)

func (pe *ParseError) WrongCreds() {
	pe.Err = true
	pe.Text = "Wrong login or password"
	pe.Type = proto.ErrInternal
}
