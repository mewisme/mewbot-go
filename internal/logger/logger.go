package logger

import (
	"fmt"
	"time"
)

const (
	reset     = "\x1b[0m"
	bold      = "\x1b[1m"
	underline = "\x1b[4m"

	brightBlack   = "\x1b[90m"
	brightRed     = "\x1b[91m"
	brightGreen   = "\x1b[92m"
	brightYellow  = "\x1b[93m"
	brightBlue    = "\x1b[94m"
	brightMagenta = "\x1b[95m"
)

type level struct {
	color string
	label string
}

var (
	levelInfo  = level{brightBlue, "i"}
	levelError = level{brightRed, "✗"}
	levelWarn  = level{brightYellow, "w"}
	levelDebug = level{brightMagenta, "d"}
	levelDone  = level{brightGreen, "✓"}
)

func log(l level, msg string) {
	ts := time.Now().Format("15:04:05")
	fmt.Printf("%s%s%s%s %s%s %s %s %s\n",
		underline, brightBlack, ts, reset, bold, l.color, l.label, reset, msg)
}

// Info logs an informational message.
func Info(format string, args ...any) { log(levelInfo, fmt.Sprintf(format, args...)) }

// Error logs an error message.
func Error(format string, args ...any) { log(levelError, fmt.Sprintf(format, args...)) }

// Warn logs a warning message.
func Warn(format string, args ...any) { log(levelWarn, fmt.Sprintf(format, args...)) }

// Debug logs a debug message.
func Debug(format string, args ...any) { log(levelDebug, fmt.Sprintf(format, args...)) }

// Done logs a success message.
func Done(format string, args ...any) { log(levelDone, fmt.Sprintf(format, args...)) }
