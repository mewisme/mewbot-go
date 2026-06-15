package main

import (
	"runtime/debug"
	"strings"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func buildVersion() string {
	if version != "dev" {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return version
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	return version
}

func buildCommit() string {
	if commit != "none" {
		return commit
	}
	if v := vcsSetting("vcs.revision"); v != "" {
		if len(v) > 7 {
			return v[:7]
		}
		return v
	}
	return commit
}

func buildDate() string {
	if date != "unknown" {
		return date
	}
	if v := vcsSetting("vcs.time"); v != "" {
		return v
	}
	return date
}

func vcsSetting(key string) string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, s := range info.Settings {
		if s.Key == key {
			return strings.TrimSpace(s.Value)
		}
	}
	return ""
}
