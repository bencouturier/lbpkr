package main

import "os"

type Config interface {
	DefaultSiteroot() string
	Siteroot() string
	RepoUrl() string
	Name() string
	Debug() bool
	RpmUpdate() bool

	// RelocateArgs returns the arguments to be passed to RPM for the repositories
	RelocateArgs(siteroot string) []string

	// RelocateFile returns the relocated file path
	RelocateFile(fname, siteroot string) string

	InitYum(*Context) error
}

// ConfigBase holds the options and defaults for the installer
type ConfigBase struct {
	siteroot  string // where to install software, binaries, ...
	repourl   string
	debug     bool
	rpmupdate bool // install/update switch
}

func (cfg *ConfigBase) Siteroot() string {
	return cfg.siteroot
}

func (cfg *ConfigBase) RepoUrl() string {
	return cfg.repourl
}

func (cfg *ConfigBase) Debug() bool {
	return cfg.debug
}

func (cfg *ConfigBase) RpmUpdate() bool {
	return cfg.rpmupdate
}

// NewConfig returns a default configuration value.
func NewConfig(cfgtype string) Config {
	switch cfgtype {
	case "atlas":
		AtlasConfig := &atlasConfig{
			ConfigBase: ConfigBase{
				siteroot: os.Getenv("MYSITEROOT"),
				repourl:  "http://atlas-computing.web.cern.ch/atlas-computing/links/reposDirectory/lcg/slc6/yum/",
			},
		}
		return AtlasConfig
	case "lhcb":
		LHCbConfig := &lhcbConfig{
			ConfigBase: ConfigBase{
				siteroot: os.Getenv("MYSITEROOT"),
				repourl:  "http://test-lbrpm.web.cern.ch/test-lbrpm",
			},
		}
		return LHCbConfig
	default:
		panic("lbpkr: unknown config [" + cfgtype + "]")
	}
	panic("unreachable")
}

// EOF
