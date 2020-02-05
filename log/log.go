package log

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"

	"github.com/bytom/bytom/config"
)

const (
	rotationTime int64 = 86400
	maxAge       int64 = 604800
)

var defaultFormatter = &logrus.TextFormatter{DisableColors: true}

func InitLogFile(config *config.Config) error {
	logPath := config.LogDir()
	if err := clearLockFiles(logPath); err != nil {
		return err
	}

	hook := newBtmHook(logPath)
	logrus.AddHook(hook)
	logrus.SetOutput(ioutil.Discard) //控制台不输出
	fmt.Printf("all logs are output in the %s directory\n", logPath)
	return nil
}

type BtmHook struct {
	logPath string
	lock    *sync.Mutex
}

func newBtmHook(logPath string) *BtmHook {
	hook := &BtmHook{lock: new(sync.Mutex)}
	hook.logPath = logPath
	return hook
}

// Write a log line to an io.Writer.
func (hook *BtmHook) ioWrite(entry *logrus.Entry) error {
	module := "general"
	if data, ok := entry.Data["module"]; ok {
		module = data.(string)
	}

	logPath := filepath.Join(hook.logPath, module)
	writer, err := rotatelogs.New(
		logPath+".%Y%m%d",
		rotatelogs.WithMaxAge(time.Duration(maxAge)*time.Second),
		rotatelogs.WithRotationTime(time.Duration(rotationTime)*time.Second),
	)
	if err != nil {
		return err
	}

	msg, err := defaultFormatter.Format(entry)
	if err != nil {
		return err
	}

	if _, err = writer.Write(msg); err != nil {
		return err
	}

	return writer.Close()
}

func clearLockFiles(logPath string) error {
	files, err := ioutil.ReadDir(logPath)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	for _, file := range files {
		if ok := strings.HasSuffix(file.Name(), "_lock"); ok {
			if err := os.Remove(filepath.Join(logPath, file.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

func (hook *BtmHook) Fire(entry *logrus.Entry) error {
	hook.lock.Lock()
	defer hook.lock.Unlock()
	return hook.ioWrite(entry)
}

// Levels returns configured log levels.
func (hook *BtmHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
