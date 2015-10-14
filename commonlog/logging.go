package commonlog

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/op/go-logging"
	"github.com/yangzhao28/utils/rotationfile"
)

const (
	CRITICAL logging.Level = logging.CRITICAL
	ERROR                  = logging.ERROR
	WARNING                = logging.WARNING
	NOTICE                 = logging.NOTICE
	INFO                   = logging.INFO
	DEBUG                  = logging.DEBUG
)

var consoleFormat = logging.MustStringFormatter(
	"[%{color}%{level:s}%{color:reset}][%{time:2006-01-02 15:04:05.000} %{shortfile}: %{longfunc}][%{id}] %{message}",
)

var fileFormat = logging.MustStringFormatter(
	"[%{level:s}][%{time:2006-01-02 15:04:05.000} %{shortfile}: %{longfunc}][%{id}] %{message}",
)

func createConsoleBackend(level logging.Level) logging.LeveledBackend {
	consoleBackend := logging.NewLogBackend(os.Stderr, "", 0)
	consoleBackendFormatter := logging.NewBackendFormatter(consoleBackend, consoleFormat)
	consoleBackendLeveled := logging.AddModuleLevel(consoleBackendFormatter)
	consoleBackendLeveled.SetLevel(level, "")
	return consoleBackendLeveled
}

func createFileBackend(fullFileName string, level logging.Level) logging.LeveledBackend {
	file := &rotationfile.Rotator{}
	file.Create(fullFileName, rotationfile.HourlyRotation)
	fileBackend := logging.NewLogBackend(file, "", 0)
	fileBackendFormatter := logging.NewBackendFormatter(fileBackend, fileFormat)
	fileBackendLeveled := logging.AddModuleLevel(fileBackendFormatter)
	fileBackendLeveled.SetLevel(level, "")
	return fileBackendLeveled
}

func NewLogger(module string, logDir string, level logging.Level) *logging.Logger {
	logger := logging.MustGetLogger(module)
	backends := make([]logging.Backend, 0)
	backends = append(backends, createConsoleBackend(level))

	absPath, err := filepath.Abs(filepath.Join(logDir, module+".log"))
	if err == nil {
		backends = append(backends, createFileBackend(absPath, level))
		fmt.Println("log file:" + absPath)
	} else {
		fmt.Println("fail to use file log: " + err.Error())
	}
	logger.SetBackend(logging.MultiLogger(backends...))
	return logger
}
