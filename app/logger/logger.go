package logger

import (
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

func InitLogging() {
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the info severity or above.
	log.SetLevel(log.InfoLevel)
}

func LogDbQueryTime(query string, args []any, err error, latency time.Duration) {
	log.WithFields(log.Fields{
		"query":           query,
		"args":            args,
		"err":             err,
		"latency_seconds": latency.Seconds(),
	}).Debug("Database query executed")
}
