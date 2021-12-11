package logger

import (
	"github.com/sirupsen/logrus"
	"io"
	"log"
	"os"
)

// InitLogger InitLoggers - начальная инициализация - лог файлы/ логгеры
// File - log file name, debug - y/n - выдача на экран одновременно
func InitLogger(File string, debug bool, loglevel int) (*logrus.Logger, error) {
	file, err := os.OpenFile(File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Printf("error creating log file %v", err)
		return nil, err
	}
	logger := logrus.New()
	logger.Formatter = new(logrus.TextFormatter)                  //default
	logger.Formatter.(*logrus.TextFormatter).DisableColors = true // remove colors
	//logger.Formatter.(*logrus.TextFormatter).DisableTimestamp = true // remove timestamp from test output
	logger.Formatter.(*logrus.TextFormatter).FullTimestamp = true
	logger.Formatter.(*logrus.TextFormatter).TimestampFormat = "2006-01-02 15:04:05"

	switch loglevel {
	case 3:
		logger.Level = logrus.DebugLevel
	case 2:
		logger.Level = logrus.WarnLevel
	case 1:
		logger.Level = logrus.InfoLevel
	}

	// if debug flag is set print all to screen and to file
	if debug {
		logger.Out = io.MultiWriter(os.Stdout, file)
	} else {
		logger.Out = io.Writer(file)
	}

	return logger, nil
}