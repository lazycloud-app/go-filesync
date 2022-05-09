package ver

import "fmt"

type (
	// AppVersion satisfies the requirements of https://semver.org/
	AppVersion struct {
		Major        uint
		MLabel       string
		Minor        uint
		Patch        uint
		ReleaseName  string
		ReleaseDate  string
		ReleaseSatus string
		Proto        Proto
	}

	// Proto represents set of message types & file processing rules that app implements
	//
	// Different protocols can have the same Ver->Mrules value if they are based on the same set of messaging rules.
	// So checking Ver->FPRules would be a good idea
	Proto struct {
		Ver          ProtoVersion
		Name         string  // Simple descr of main feature (e.g. 'Zero-knowledge' or 'Extended hash')
		Author       string  // Website / name / email / company
		Docs         string  // Link to documentation
		Compatible   []Proto // Versions that have been tested for compatibility
		InCompatible []Proto // Versions that are definitely incompatible
	}

	ProtoVersion struct {
		MRules     uint // Messaging rules
		MRulesExp  string
		FPRules    uint // File processing rules
		FPRulesExp string
	}
)

func (v AppVersion) String() string {
	return fmt.Sprintf(`%d.%d.%d-%s "%s" (%s) Branch:%s`, v.Major, v.Minor, v.Patch, v.MLabel, v.ReleaseName, v.ReleaseDate, v.ReleaseSatus)
}

func (v Proto) String() string {
	return fmt.Sprintf("%d.%d (%s) by %s", v.Ver.MRules, v.Ver.FPRules, v.Name, v.Author)
}

func (v Proto) GetDocs() string {
	return v.Docs
}

func (v ProtoVersion) String() string {
	return fmt.Sprintf("%d(%s) %d(%s)", v.MRules, v.MRulesExp, v.FPRules, v.FPRulesExp)
}

// FullCompartion should be used to avoid any differences between versions.
// It's the most desirable way of comparing, to avoid any funny situations
func (v Proto) FullComparsion(c Proto) bool {
	if v.Ver == c.Ver && v.Name == c.Name {
		return true
	}
	for _, icv := range c.InCompatible {
		if v.Ver == icv.Ver && v.Name == icv.Name {
			return false
		}
	}
	for _, cv := range c.Compatible {
		if v.Ver == cv.Ver && v.Name == cv.Name {
			return true
		}
	}
	return false
}
