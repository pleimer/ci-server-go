package logging

import (
	"bytes"
	"fmt"
	"os"
	"time"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

func (l LogLevel) String() string {
	return [...]string{"DEBUG", "INFO", "WARN", "ERROR"}[l]
}

type writer struct {
	target  string
	logfile *os.File
}

func (w *writer) Write(message string) error {
	switch w.target {
	case "console":
		fmt.Print(message)
		break
	default:
		var err error
		if w.logfile == nil {
			w.logfile, err = os.OpenFile(w.target, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
			if err != nil {
				return err
			}
		}
		_, err = w.logfile.WriteString(message)
		if err != nil {
			return err
		}
		break
	}
	return nil
}

func (w *writer) Close() error {
	if w.logfile != nil {
		return w.logfile.Close()
	}
	return nil
}

type Logger struct {
	Level     LogLevel
	Timestamp bool
	metadata  map[string]interface{}
	writer    *writer
}

func NewLogger(level LogLevel, target string) *Logger {
	var logger Logger
	logger.Level = level
	logger.Timestamp = false
	logger.metadata = make(map[string]interface{})

	logger.writer = &writer{
		target: target,
	}

	return &logger
}

func (l *Logger) Destroy() error {
	return l.writer.Close()
}

func (l *Logger) Metadata(metadata map[string]interface{}) {
	l.metadata = metadata
}

func (l *Logger) formatMetadata() (string, error) {
	//var build strings.Builder
	// Note: we need to support go-1.9.2 because of CentOS7
	var build bytes.Buffer
	if len(l.metadata) > 0 {
		joiner := ""
		for key, item := range l.metadata {
			_, err := fmt.Fprintf(&build, "%s%s: %v", joiner, key, item)
			if err != nil {
				return build.String(), err
			}
			if len(joiner) == 0 {
				joiner = ", "
			}
		}
	}
	// clear metadata for next use
	l.metadata = make(map[string]interface{})
	return build.String(), nil
}

func (l *Logger) writeRecord(level LogLevel, message string) error {
	metadata, err := l.formatMetadata()
	if err != nil {
		return err
	}

	//var build strings.Builder
	// Note: we need to support go-1.9.2 because of CentOS7
	var build bytes.Buffer
	if l.Timestamp {
		_, err = build.WriteString(time.Now().Format("2006-01-02 15:04:05 "))
	}

	_, err = build.WriteString(fmt.Sprintf("[%s] ", level))
	if err != nil {
		return nil
	}
	_, err = build.WriteString(message)
	if err != nil {
		return nil
	}
	if len(metadata) > 0 {
		_, err = build.WriteString(fmt.Sprintf(" [%s]", metadata))
		if err != nil {
			return nil
		}
	}
	_, err = build.WriteString("\n")
	if err != nil {
		return nil
	}
	err = l.writer.Write(build.String())
	return err
}

func (l *Logger) Debug(message string) error {
	if l.Level == DEBUG {
		return l.writeRecord(DEBUG, message)
	}
	return nil
}

func (l *Logger) Info(message string) error {
	if l.Level <= INFO {
		return l.writeRecord(INFO, message)
	}
	return nil
}

func (l *Logger) Warn(message string) error {
	if l.Level <= WARN {
		return l.writeRecord(WARN, message)
	}
	return nil
}

func (l *Logger) Error(message string) error {
	if l.Level <= ERROR {
		return l.writeRecord(ERROR, message)
	}
	return nil
}
