package events

import (
	"log"

	"github.com/lazybark/go-pretty-code/logs"
	"go.uber.org/zap"
)

// NewProcessor creates new processor
func NewStandartLogsProcessor(logfile string, isVerbose bool) (p *Processor) {
	p = new(Processor)
	p.ec = make(chan Event)
	p.sc = make(chan bool)
	p.IsVerbose = isVerbose

	p.StandartLogs(logfile)

	return
}

// StandartLogs starts StandartLogsProcessor
func (p *Processor) StandartLogs(logfile string) {
	go p.StandartLogsProcessor(logfile)
}

// StandartLogsProcessor log to both file + console
//
// If event is verbose and p.IsVerbose = false, StandartLogsProcessor will skip that event
func (p *Processor) StandartLogsProcessor(logpath string) {

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
