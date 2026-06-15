package main

import (
	"runtime/debug"
	"strings"
)

const modulePath = "github.com/mewisme/mewbot-go"

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
	if v := moduleVersion(info); v != "" {
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

func moduleVersion(info *debug.BuildInfo) string {
	if info.Main.Path == modulePath {
		if v := normalizeVersion(info.Main.Version); v != "" {
			return v
		}
	}
	for _, dep := range info.Deps {
		if dep.Path == modulePath {
			if v := normalizeVersion(dep.Version); v != "" {
				return v
			}
		}
	}
	return ""
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || v == "(devel)" {
		return ""
	}
	if i := strings.IndexByte(v, '+'); i >= 0 {
		v = v[:i]
	}
	return v
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
