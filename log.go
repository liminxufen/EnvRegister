package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

type Logger struct {
	Name string
}

type config struct {
	FilePath string
	Debug    bool
	*os.File
}

var currentFile *os.File

var Config = &config{
	Debug:    false,
	FilePath: "/tmp/tusk.log",
}

func NewLogger(name string) (logger *Logger) {
	return &Logger{Name: name}
}

func (logger *Logger) Info(v ...interface{}) {
	logger.commonLog("INFO", v...)
}

func (logger *Logger) Infof(format string, v ...interface{}) {
	logger.commonLogf("INFO", format, v...)
}

func (logger *Logger) Warn(v ...interface{}) {
	logger.commonLog("WARN", v...)
}

func (logger *Logger) Debug(v ...interface{}) {
	logger.commonLog("DEBUG", v...)
}

func (logger *Logger) Debugf(format string, v ...interface{}) {
	logger.commonLogf("DEBUG", format, v...)
}

func (logger *Logger) Warnf(format string, v ...interface{}) {
	logger.commonLogf("WARN", format, v...)
}

func (logger *Logger) Error(v ...interface{}) {
	logger.commonLog("ERROR", v...)
}

func (logger *Logger) Errorf(format string, v ...interface{}) {
	logger.commonLogf("ERROR", format, v...)
}

func (logger *Logger) Fatal(v ...interface{}) {
	logger.Error(v...)
	os.Exit(1)
}

func (logger *Logger) Fatalf(format string, v ...interface{}) {
	logger.Errorf(format, v...)
	os.Exit(1)
}

func (logger *Logger) commonLog(logType string, v ...interface{}) {
	logStr := fmt.Sprintln(v...)
	logStr = fmt.Sprintf("[%s][%s] %s", logType, logger.Name, logStr)
	writeLog(logStr)
}

func (logger *Logger) commonLogf(logType string, format string, v ...interface{}) {
	logStr := fmt.Sprintf(format, v...)
	logStr = fmt.Sprintf("[%s][%s] %s", logType, logger.Name, logStr)
	writeLog(logStr)
}

func Init(logFilePath string, debug bool) {
	Config.FilePath = logFilePath
	Config.Debug = debug
	Config.File = nil
	log.SetFlags(log.Ldate | log.Ltime)
	return
}

func getLogFilePath() (path string) {
	if len(Config.FilePath) == 0 {
		return
	}
	tmps := strings.Split(Config.FilePath, ".")
	if tmps[len(tmps)-1] == "log" {
		tmps = tmps[:len(tmps)-1]
	}
	dateStr := time.Now().Format("20060102")
	tmps = append(tmps, []string{dateStr, "log"}...)
	return strings.Join(tmps, ".")
}

func writeLog(logStr string) (err error) {
	err = resetOutputIfNeed()
	if err != nil {
		fmt.Println("writeLog ERROR, resetOutput: ", err)
		return
	}
	var (
		fp  string
		lno int
	)

	_, fp, lno, ok := runtime.Caller(3)
	if ok {
		sfp := fp
		for i := len(fp) - 1; i > 0; i-- {
			if fp[i] == '/' {
				sfp = fp[i+1:]
				break
			}
		}
		fp = sfp
	} else {
		fp = "???"
		lno = 0
	}
	logStr = fmt.Sprintf("%s:%d %s", fp, lno, logStr)

	log.Print(logStr)
	return
}

func resetOutputIfNeed() (err error) {
	needReset := false
	logFilePath := getLogFilePath()
	if Config.File == nil {
		needReset = true
	} else {
		fileInfo, statErr := Config.File.Stat()
		if statErr != nil {
			needReset = true
		} else if !strings.HasSuffix(logFilePath, fileInfo.Name()) {
			needReset = true
		}
	}
	if needReset {
		oldFile := Config.File
		newFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.ModePerm)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		defer closeFile(oldFile)
		Config.File = newFile
		var output io.Writer
		if Config.Debug {
			output = io.MultiWriter(os.Stderr, Config.File)
		} else {
			output = Config.File
		}
		log.SetOutput(output)
	}
	return
}

func closeFile(file *os.File) {
	if file == nil {
		return
	}
	err := file.Close()
	if err != nil {
		fmt.Println("ERROR AT CLOSING LOG FILE:", err.Error())
	}
	return
}
