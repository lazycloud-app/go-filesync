package syncev

//IE represents internal event, error or warning. It's meant to be used by client/server error processors only
//and not to be exported via API to other mods.
//Avoid using this type in methods that can be reused in other projetcs as it will make their use more complicated.
//
//Better to wrap builtin error instead.
type IE struct {
	Err      error
	IsErr    bool
	EvSource EvSource //One ofre-defined sources to log/treat error correctly
	Log      bool     //Error should be logged
	Level    EvLevel  //Errors are treated accordingly to that level
}

func (ie IE) IsError() bool {
	return ie.IsErr
}

func (ie IE) Error() string {
	return ie.Err.Error()
}

func (ie IE) Source() string {
	return ie.EvSource.String()
}

func (ie *IE) NoLog() {
	ie.Log = false
}

//EvSource represents possible error sources to help debugging the app
type EvSource int

const (
	EvSourceUnknown EvSource = iota

	EvSourceConfig
	EvSourceStartup

	EvSourceIllegal
)

func (es EvSource) String() string {
	sourceNames := [...]string{"Unknown event source", "Configurator", "Startup", "Illegal event source"}
	if EvSourceUnknown > es || es > EvSourceIllegal {
		return "Unknown event source"
	}
	return sourceNames[es]
}

//EvLevel regulates the way events should be treated by event processors
type EvLevel int

const (
	EvLevelUnknown EvLevel = iota

	EvLevelInfo
	EvLevelWarning
	EvLevelError
	EvLevelCritical
	EvLevelFatal

	EvLevelIllegal
)

func (el EvLevel) String() string {
	levelNames := [...]string{"", "INFO", "WARNING", "ERROR", "CRITICAL", "FATAL", ""}
	if EvLevelUnknown > el || el > EvLevelIllegal {
		return "UNKNOWN"
	}
	return levelNames[el]
}
