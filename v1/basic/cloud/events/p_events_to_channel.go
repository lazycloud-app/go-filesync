package events

import (
	"fmt"
	"log"

	"github.com/lazybark/go-pretty-code/logs"
	"go.uber.org/zap"
)

// NewProcessor creates new processor
func NewEventsToChannelProcessor(logfile string, isVerbose bool, c chan string) (p *Processor) {
	p = new(Processor)
	p.ec = make(chan Event)
	p.sc = make(chan bool)
	p.IsVerbose = isVerbose
	p.extc = c

	p.EventsToChannel(logfile)

	return
}

// StandartLogs starts StandartLogsProcessor
func (p *Processor) EventsToChannel(logfile string) {
	go p.EventsToChannelProcessor(logfile)
}

// StandartLogsProcessor log to both file + console
//
// If event is verbose and p.IsVerbose = false, StandartLogsProcessor will skip that event
func (p *Processor) EventsToChannelProcessor(logpath string) {

	// Connect Logger
	ld, err := logs.Double(logpath, false, zap.InfoLevel)
	if err != nil {
		log.Fatal("[EventProcessor] error -> unable to make logger: ", err)
		return
	}
	defer ld.Info("[EventProcessor] warning -> events channel closed")
	defer func() { p.sc <- true }()

	for {
		select {
		case event, ok := <-p.ec:
			if !ok {
				return
			}
			if !event.Level.CheckEventLevel() {
				ld.Error("[EventProcessor] error -> illegal event level: ", event.Level.String())
				continue
			}
			// Act according to logging config (mainly for base server needs)
			if !p.IsVerbose && event.Verbose {
				continue
			} else {
				p.Log(ld, event)
			}
		case stop, ok := <-p.sc:
			if !ok || stop {
				return
			}
		}
	}
}

func (p *Processor) ToChannel(event Event) {
	err, isErr := event.Data.(error)
	if isErr {
		if event.Level < Warn {
			p.extc <- "[EventProcessor] error -> event type & level mismatch: wanted Warn or above, got less"
			return
		} else {
			p.extc <- fmt.Sprintf("[%s] %v", event.Source, err)
		}
	}

	text, isText := event.Data.(string)
	if isText {
		p.extc <- fmt.Sprintf("[%s] %s", event.Source, text)
	}

	if !isErr && !isText {
		p.extc <- "[EventProcessor] error -> wrong event payload type: 'error' or 'string' only"
		return
	}
}
