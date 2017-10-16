package main

import (
	"github.com/bytom/cmd/bytomd/commands"
	"github.com/sirupsen/logrus"
	"github.com/tendermint/tmlibs/cli"
	"os"
	"path"
	"runtime"
	"strings"
)

type ContextHook struct{}

func (hook ContextHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (hook ContextHook) Fire(entry *logrus.Entry) error {
	pc := make([]uintptr, 3, 3)
	cnt := runtime.Callers(6, pc)

	for i := 0; i < cnt; i++ {
		fu := runtime.FuncForPC(pc[i] - 1)
		name := fu.Name()
		if !strings.Contains(name, "github.com/Sirupsen/logrus") {
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
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
}

func main() {
	logrus.AddHook(ContextHook{})
	cmd := cli.PrepareBaseCmd(commands.RootCmd, "TM", os.ExpandEnv("./.bytomd"))
	cmd.Execute()
}
