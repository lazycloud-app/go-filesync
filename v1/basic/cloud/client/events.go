package client

import "github.com/lazycloud-app/go-filesync/v1/basic/cloud/events"

// EventType returns event type according to go-filesync/cloud/events Events model
func EventType(name string) events.Level {
	if name == "info" {
		return events.InfoGreen
	} else if name == "green" {
		return events.InfoGreen
	} else if name == "cyan" {
		return events.InfoCyan
	} else if name == "red" {
		return events.InfoRed
	} else if name == "magenta" {
		return events.InfoMagenta
	} else if name == "yellow" {
		return events.InfoYellow
	} else if name == "backred" {
		return events.InfoBackRed
	} else if name == "fatal" {
		return events.Fatal
	} else if name == "warn" {
		return events.Warn
	} else if name == "error" {
		return events.Error
	}
	return events.UnknownLevel
}
