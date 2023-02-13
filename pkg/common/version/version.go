package version

import "fmt"

var (
	version   = "unset"
	commit    = "unset"
	buildTime = "unset"
)

func Format() string {
	return fmt.Sprintf("version '%s', commit '%s', build time '%s'", version, commit, buildTime)
}

func SetVersion(_version, _commit, _buildTime string) {
	version = _version
	commit = _commit
	buildTime = _buildTime
}
