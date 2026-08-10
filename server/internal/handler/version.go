package handler

import (
	"log"
	"os/exec"
	"runtime/debug"
	"strings"
)

var (
	Version   = "1.0.0"
	GitCommit = ""
	BuildTime = ""
	Dirty     = false
)

func init() {
	info, ok := debug.ReadBuildInfo()
	if ok {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				if len(setting.Value) > 7 {
					GitCommit = setting.Value[:7]
				} else {
					GitCommit = setting.Value
				}
			case "vcs.time":
				BuildTime = setting.Value
			case "vcs.modified":
				Dirty = setting.Value == "true"
			}
		}
	}

	if GitCommit == "" {
		GitCommit = gitRevParseShort()
	}
	if BuildTime == "" {
		BuildTime = gitCommitTime()
	}
}

func gitRevParseShort() string {
	out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		log.Printf("Warning: failed to get git commit: %v", err)
		return ""
	}
	return strings.TrimSpace(string(out))
}

func gitCommitTime() string {
	out, err := exec.Command("git", "log", "-1", "--format=%cI").Output()
	if err != nil {
		log.Printf("Warning: failed to get git commit time: %v", err)
		return ""
	}
	return strings.TrimSpace(string(out))
}
