package logger

import (
	"fmt"
	"os"
	"time"
)

var logFile *os.File

func init() {
	var err error
	logFile, err = os.OpenFile("gopy_errors.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
}

func Log(msg string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s\n", timestamp, msg)
	logFile.WriteString(logEntry)
}

func Close() {
	logFile.Close()
}
