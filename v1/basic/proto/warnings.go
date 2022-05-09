package proto

type (
	WarningType int

	Warning struct {
		Type      WarningType
		Explained string
	}
)

const (
	warnings_start WarningType = iota

	WarnUnsupportedFunction
	WarnMaintenance
	WarnProtocolChange
	WarnPossibleErrors
	WarnUnknownObjects
	WarnConnectionLimitReached

	warnings_end
)

func (e WarningType) Check() bool {
	if warnings_start > e && e > warnings_end {
		return false
	}
	return true
}

func (w WarningType) String() string {
	if !w.Check() {
		return "Unknown warning"
	}
	return [...]string{"Unknown warning", "WarnUnsupportedFunction", "WarnMaintenance", "WarnProtocolChange", "WarnPossibleErrors", "WarnUnknownObjects", "WarnConnectionLimitReached", "Unknown waning"}[w]
}
