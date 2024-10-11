package tlog

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"
)

type Level uint8

const (
	DEBUG Level = 1
	INFO  Level = 2
	WARN  Level = 3
	ERROR Level = 4
)

func (l Level) String() string {
	switch l {
	case DEBUG:
		return "debug"
	case INFO:
		return "info"
	case WARN:
		return "warn"
	case ERROR:
		return "error"
	default:
		return fmt.Sprintf("level(%d)", l)
	}
}

type Logger interface {
}

type Field struct {
	Key   string
	Value any
}

func (f *Field) String() string {
	return fmt.Sprintf("%s=%+v", f.Key, f.Value)
}

func Any(k string, v any) *Field {
	return &Field{Key: k, Value: v}
}

func Debug(msg string, fields ...*Field) {
	write(context.TODO(), DEBUG, msg, fields...)
}

func Debugc(ctx context.Context, msg string, fields ...*Field) {
	write(ctx, DEBUG, msg, fields...)
}

func Info(msg string, fields ...*Field) {
	write(context.TODO(), INFO, msg, fields...)
}

func Infoc(ctx context.Context, msg string, fields ...*Field) {
	write(ctx, INFO, msg, fields...)
}

func Warn(msg string, fields ...*Field) {
	write(context.TODO(), WARN, msg, fields...)
}

func Warnc(ctx context.Context, msg string, fields ...*Field) {
	write(ctx, WARN, msg, fields...)
}

func Error(msg string, fields ...*Field) {
	write(context.TODO(), ERROR, msg, fields...)
}

func Errorc(ctx context.Context, msg string, fields ...*Field) {
	write(ctx, ERROR, msg, fields...)
}

func Flush() {
	log.writer.Close()
}

func write(ctx context.Context, lv Level, msg string, fields ...*Field) {
	if log.level > lv {
		return
	}
	s := fmt.Sprintf(defaultLogLayout, log.timeFormat(time.Now()), lv)
	for _, f := range log.fields {
		s = s + f.String() + "|"
	}
	s = s + msg
	for i, f := range fields {
		if i > 0 {
			s = s + " "
		} else {
			s = s + "|"
		}
		s = s + f.String()
	}
	log.writer.Write([]byte(s + "\n"))
}

type TimeFormat func(t time.Time) string

func defaultTimeFormat(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.000")
}

type logger struct {
	timeFormat TimeFormat
	fields     []*Field
	level      Level
	writer     io.WriteCloser
}

var log *logger

const defaultLogLayout = "%s|%s|"

func init() {
	log = &logger{
		timeFormat: defaultTimeFormat,
		fields:     make([]*Field, 0),
		level:      DEBUG,
		writer:     os.Stdout,
	}
	log.fields = append(log.fields, Any("pid", os.Getpid()))
}
