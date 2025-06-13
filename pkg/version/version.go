package version

import (
	"fmt"
	"runtime"
)

var (
	Version   = "dev"     // -ldflags "-X pkg/version.Version=v1.2.3"
	GitCommit = "unknown" // -ldflags "-X pkg/version.GitCommit=abc123"
	BuildTime = "unknown" // -ldflags "-X pkg/version.BuildTime=2024-01-01T12:00:00Z"
	GoVersion = runtime.Version()
	Platform  = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
)

type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

func Get() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
		GoVersion: GoVersion,
		Platform:  Platform,
	}
}

func (i Info) String() string {
	return fmt.Sprintf("blobcast %s (commit: %s, built: %s, go: %s, platform: %s)",
		i.Version, i.GitCommit, i.BuildTime, i.GoVersion, i.Platform)
}
