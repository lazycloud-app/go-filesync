package proto

import (
	"github.com/lazycloud-app/go-fsp-proto/ver"
)

var (
	ClientVersion = ver.AppVersion{
		Major:        0,
		MLabel:       "alpha",
		Minor:        0,
		Patch:        0,
		ReleaseName:  "Test client",
		ReleaseDate:  "2022.05.05",
		ReleaseSatus: "dev",
		Proto:        ProtoVer,
	}
	ClientVersionLabel = "unstable pre-alpha"

	ServerVersion = ver.AppVersion{
		Major:        0,
		MLabel:       "alpha",
		Minor:        0,
		Patch:        0,
		ReleaseName:  "Test server",
		ReleaseDate:  "2022.05.05",
		ReleaseSatus: "dev",
		Proto:        ProtoVer,
	}
	ServerVersionLabel = "unstable pre-alpha"

	ProtoVer = ver.Proto{
		Ver:    RulesVer,
		Name:   "Basic",
		Author: "Lazycloud soft",
		Docs:   "https://github.com/lazycloud-app",
	}

	RulesVer = ver.ProtoVersion{
		MRules:     0,
		MRulesExp:  "Basic messaging rules",
		FPRules:    0,
		FPRulesExp: "Basic file processing rules",
	}
)
