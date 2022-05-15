package syncerr

//IE represents internal error or warning. It's meant to be used by client/server error processors only
//and not to be exported via API to other mods.
//Avoid using this type in methods that can be reused in other projetcs as it will make their use more complicated.
//
//Better to wrap builtin error instead.
type IE struct {
	Err       error
	ErrSource ErrSource //One ofre-defined sources to log/treat error correctly
	Log       bool      //Error should be logged
	Level     ErrLevel  //Errors are treated accordingly to that level
}

func (ie IE) Error() string {
	return ie.Err.Error()
}

func (ie IE) Source() string {
	return ie.ErrSource.String()
}

func (ie *IE) NoLog() {
	ie.Log = false
}

//ErrSource represents possible error sources to help debugging the app
type ErrSource int

const (
	ErrSourceUnknown ErrSource = iota

	ErrSourceConfig

	ErrSourceIllegal
)

func (es ErrSource) String() string {
	sourceNames := [...]string{"Unknown err source", "Configurator", "Illegal err source"}
	if ErrSourceUnknown > es || es > ErrSourceIllegal {
		return "Unknown err source"
	}
	return sourceNames[es]
}

//ErrLevel regulates the way errors should be treated by error processors
type ErrLevel int

const (
	ErrLevelUnknown ErrLevel = iota

	ErrLevelInfo
	ErrLevelWarning
	ErrLevelError
	ErrLevelCritical
	ErrLevelFatal

	ErrLevelIllegal
)

func (el ErrLevel) String() string {
	levelNames := [...]string{"", "INFO", "WARNING", "ERROR", "CRITICAL", "FATAL", ""}
	if ErrLevelUnknown > el || el > ErrLevelIllegal {
		return "UNKNOWN"
	}
	return levelNames[el]
}
