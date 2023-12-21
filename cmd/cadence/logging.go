package main

import (
	logger "github.com/sirupsen/logrus"
)

type UTCFormatter struct {
	logger.Formatter
}

func (u UTCFormatter) Format(e *logger.Entry) ([]byte, error) {
	e.Time = e.Time.UTC()
	return u.Formatter.Format(e)
}
