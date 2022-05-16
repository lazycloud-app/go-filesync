package syncev

//New returns asserts error source and level, then calls to NewErr
func New(t string, s string, l bool, lev string) IE {
	source := AssetSource(s)
	level := AssetLevel(lev)

	return NewEv(t, source, l, level)
}

//NewEv returns new IE object marked as error
func NewEv(t string, s EvSource, l bool, lev EvLevel) IE {
	return IE{
		Text:     t,
		EvSource: s,
		Log:      l,
		Level:    lev,
	}
}

//AssetSource returns error source based on provided string.
//Its use is logical in cases where ErrSource const names may change (depending packages will have no problems in that case)
func AssetSource(s string) EvSource {
	if s == "startup" {
		return EvSourceStartup
	}
	return EvSourceUnknown
}

//AssetLevel returns error level based on provided string.
//Its use is logical in cases where ErrLevel const names may change (depending packages will have no problems in that case)
func AssetLevel(s string) EvLevel {
	if s == "fatal" {
		return EvLevelFatal
	}
	return EvLevelUnknown
}
