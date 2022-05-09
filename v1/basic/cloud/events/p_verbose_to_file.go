package events

import (
	"log"

	"github.com/lazybark/go-pretty-code/logs"
	"go.uber.org/zap"
)

// NewProcessor creates new processor
func NewVerboseToLogsOnlyProcessor(logfile string, isVerbose bool) (p *Processor) {
	p = new(Processor)
	p.ec = make(chan Event)
	p.sc = make(chan bool)
	p.IsVerbose = isVerbose

	p.VerboseToLogsOnly(logfile)

	return
}

// VerboseToLogsOnly starts VerboseToLogsOnlyProcessor
func (p *Processor) VerboseToLogsOnly(logfile string) {
	go p.VerboseToLogsOnlyProcessor(logfile)
}

// VerboseToLogsOnlyProcessor logs to both file + console
//
// If event is verbose and p.IsVerbose = false, VerboseToLogsOnlyProcessor will put that event to file only
func (p *Processor) VerboseToLogsOnlyProcessor(logpath string) {

	// Connect Logger
	ld, err := logs.Double(logpath, false, zap.InfoLevel)
	if err != nil {
		log.Fatal("[EventProcessor] error -> unable to make double logger: ", err)
		return
	}
	lf, err := logs.FileOnly(logpath, false, zap.InfoLevel)
	if err != nil {
		log.Fatal("[EventProcessor] error -> unable to make file logger: ", err)
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
				// Log to file all verbose events
				p.Log(lf, event)
				continue
			} else {
				// Log to file + console all important events
				p.Log(ld, event)
				continue
			}

		case stop, ok := <-p.sc:
			if !ok || stop {
				return
			}
		}
	}
}
