package events

import (
	"fmt"

	"github.com/lazybark/go-pretty-code/logs"
)

type (
	Processor struct {
		// Internal channel for events
		ec chan (Event)
		// External channel to recieve 'stop'
		sc chan (bool)
		// External channel to send stringed event data
		extc chan (string)
		// IsVerbose indicates whether this instance will process events marked as Verbose
		IsVerbose bool
	}

	Event struct {
		// Level indicates the way to process Event
		Level Level
		// Source in most cases will be printed out in logs at the beginning of each record
		Source string
		// Verbose is useful to avoid unnecessary fuzz in logs (in case sender don't filter events on the source)
		Verbose bool
		// Data should contain payload of error or string
		Data interface{}
	}

	EventProcessor interface {
		// Send message that should be processed at any circumstances
		Send(Level, string, interface{})
		// Send message that should be ignored in case verbosity is set to false
		SendVerbose(Level, string, interface{})
		// Signal EventProcessor that there will be no events anymore
		Close()
	}
)

// SendVerbose sends message marked as non-verbose that will be treated accordingly to IsVerbose property of the Processor
func (p *Processor) Send(l Level, es string, data interface{}) {
	p.ec <- Event{Level: l, Source: es, Verbose: false, Data: data}
}

// SendVerbose sends message marked as verbose that will be treated accordingly to IsVerbose property of the Processor
func (p *Processor) SendVerbose(l Level, es string, data interface{}) {
	p.ec <- Event{Level: l, Source: es, Verbose: true, Data: data}
}

func (p *Processor) Close() {
	p.sc <- true
	<-p.sc
}

func (p *Processor) Log(ld *logs.Logger, event Event) {
	err, isErr := event.Data.(error)
	if isErr {
		if event.Level < Warn {
			ld.Error("[EventProcessor] error -> event type & level mismatch: wanted Warn or above, got less")
			return
		} else if event.Level == Warn {
			ld.Warn(fmt.Sprintf("[%s] ", event.Source), err)
		} else if event.Level == Error {
			ld.Error(fmt.Sprintf("[%s] ", event.Source), err)
		} else if event.Level == Fatal {
			ld.Fatal(fmt.Sprintf("[%s] ", event.Source), err)
		} else {
			ld.Error(fmt.Sprintf("((%s level event))[%s] ", event.Level.String(), event.Source), err)
		}
	}

	text, isText := event.Data.(string)
	if isText {
		if event.Level == Info {
			ld.Info(fmt.Sprintf("[%s] ", event.Source), text)
		} else if event.Level == InfoCyan {
			ld.InfoCyan(fmt.Sprintf("[%s] ", event.Source), text)
		} else if event.Level == InfoBackRed {
			ld.InfoBackRed(fmt.Sprintf("[%s] ", event.Source), text)
		} else if event.Level == InfoRed {
			ld.InfoRed(fmt.Sprintf("[%s] ", event.Source), text)
		} else if event.Level == InfoYellow {
			ld.InfoYellow(fmt.Sprintf("[%s] ", event.Source), text)
		} else if event.Level == InfoGreen {
			ld.InfoGreen(fmt.Sprintf("[%s] ", event.Source), text)
		} else if event.Level == InfoMagenta {
			ld.InfoMagenta(fmt.Sprintf("[%s] ", event.Source), text)
		} else if event.Level == Warn {
			ld.Warn(fmt.Sprintf("[%s] ", event.Source), text)
		} else if event.Level == Error {
			ld.Error(fmt.Sprintf("[%s] ", event.Source), text)
		} else if event.Level == Fatal {
			ld.Fatal(fmt.Sprintf("[%s] ", event.Source), text)
		} else {
			ld.Info(fmt.Sprintf("((%s level event))[%s] ", event.Level.String(), event.Source), text)
		}
	}

	if !isErr && !isText {
		ld.Error("[EventProcessor] error -> wrong event payload type: 'error' or 'string' only")
		return
	}
}
