package main

import (
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"runtime"
	"strings"
)

type ContextHook struct{}

func (hook ContextHook) Levels() []log.Level {
	return log.AllLevels
}

func (hook ContextHook) Fire(entry *log.Entry) error {
	pc := make([]uintptr, 3, 3)
	cnt := runtime.Callers(6, pc)

	for i := 0; i < cnt; i++ {
		fu := runtime.FuncForPC(pc[i] - 1)
		name := fu.Name()
		if !strings.Contains(name, "github.com/Sirupsen/log") {
			file, line := fu.FileLine(pc[i] - 1)
			entry.Data["file"] = path.Base(file)
			entry.Data["func"] = path.Base(name)
			entry.Data["line"] = line
			break
		}
	}
	return nil
}

func init() {
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})

	// If environment variable BYTOM_DEBUG is not empty,
	// then add the hook to logrus and set the log level to DEBUG
	if os.Getenv("BYTOM_DEBUG") != "" {
		log.AddHook(ContextHook{})
		log.SetLevel(log.DebugLevel)
	}
}
