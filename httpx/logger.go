package ehttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type Logger interface {
	Warn(args ...any)
	Warnf(format string, args ...any)
	Info(args ...any)
	Infof(format string, args ...any)
	Error(args ...any)
	Errorf(format string, args ...any)
	Fatal(args ...any)
	Fatalf(format string, args ...any)
}

// Fields log fields
type Fields map[string]any

type (
	FieldKey string
	FieldMap map[FieldKey]string
)

func (f FieldMap) resolve(key FieldKey) string {
	if k, ok := f[key]; ok {
		return k
	}
	return string(key)
}

// Level log level
type Level uint8

const (
	PanicLevel Level = iota
	FatalLevel
	ErrorLevel
	InfoLevel
	WarnLevel
)

type stdOutLogger struct {
	stdOutLog io.Writer
	stdErrLog io.Writer

	Formatter Formatter
	entryPool sync.Pool

	fieldResolver FieldMap
}

func NewStdLogger() Logger {
	logger := &stdOutLogger{
		stdOutLog: os.Stdout,
		stdErrLog: os.Stderr,
		Formatter: NewJSONFormatter(),
		fieldResolver: map[FieldKey]string{
			FieldKeyLevel: "@level",
			FieldKeyTime:  "@time",
			FieldKeyMsg:   "@msg",
		},
	}
	logger.entryPool.New = func() any {
		return logger.newEntry()
	}
	return logger
}

func (log *stdOutLogger) FieldResolver(resolver FieldMap) {
	log.fieldResolver = resolver
}

func ParseLevelText(level string) (Level, error) {
	switch strings.ToLower(level) {
	case "panic":
		return PanicLevel, nil
	case "fatal":
		return FatalLevel, nil
	case "error":
		return ErrorLevel, nil
	case "warning":
		return WarnLevel, nil
	case "info":
		return InfoLevel, nil
	}

	var l Level
	return l, fmt.Errorf("not a valid Level: %q", level)
}

func ParseLevel(level Level) string {
	switch level {
	case PanicLevel:
		return "panic"
	case FatalLevel:
		return "fatal"
	case ErrorLevel:
		return "error"
	case WarnLevel:
		return "warning"
	case InfoLevel:
		return "info"
	}
	return ""
}

func (log *stdOutLogger) Warn(args ...any) {
	logger := log.getEntry(log.stdOutLog)
	logger.log(WarnLevel, "", args)
}

func (log *stdOutLogger) Warnf(format string, args ...any) {
	log.getEntry(log.stdOutLog).log(WarnLevel, format, args...)
}

func (log *stdOutLogger) Info(args ...any) {
	log.getEntry(log.stdOutLog).log(InfoLevel, "", args...)
}

func (log *stdOutLogger) Infof(format string, args ...any) {
	log.getEntry(log.stdOutLog).log(InfoLevel, format, args...)
}

func (log *stdOutLogger) Error(args ...any) {
	log.getEntry(log.stdErrLog).log(ErrorLevel, "", args...)
}

func (log *stdOutLogger) Errorf(format string, args ...any) {
	log.getEntry(log.stdErrLog).log(ErrorLevel, format, args...)
}

func (log *stdOutLogger) Fatal(args ...any) {
	log.getEntry(log.stdErrLog).log(FatalLevel, "", args...)
}

func (log *stdOutLogger) Fatalf(format string, args ...any) {
	log.getEntry(log.stdErrLog).log(FatalLevel, format, args...)
}

const (
	defaultTimestampFormat          = time.RFC3339
	FieldKeyLevel          FieldKey = "level"
	FieldKeyTime           FieldKey = "ts"
	FieldKeyMsg            FieldKey = "msg"
)

type Entry struct {
	Logger *stdOutLogger
	Out    io.Writer
	buf    *bytes.Buffer

	level Level
	data  Fields
}

func (log *stdOutLogger) newEntry() *Entry {
	return &Entry{
		Logger: log,
	}
}

func (log *stdOutLogger) getEntry(out io.Writer) *Entry {
	entry := log.entryPool.Get().(*Entry)
	entry.Out = out
	entry.buf = defaultBufPool.Get()
	return entry
}

func (entry *Entry) releaseEntry() {
	entry.reset()
	entry.Logger.entryPool.Put(entry)
}

func (entry *Entry) reset() {
	entry.Out = nil
	entry.data = nil
	entry.level = InfoLevel
	defaultBufPool.Put(entry.buf)
	entry.buf = nil
}

func (entry *Entry) log(level Level, format string, args ...any) {
	entry.level = level
	fieldResolver := entry.Logger.fieldResolver
	if entry.data == nil {
		entry.data = make(Fields, 6)
	}
	defer entry.releaseEntry()
	entry.data[fieldResolver.resolve(FieldKeyLevel)] = ParseLevel(entry.level)
	entry.data[fieldResolver.resolve(FieldKeyTime)] = time.Now().Format(defaultTimestampFormat)
	entry.data[fieldResolver.resolve(FieldKeyMsg)] = entry.parseMessage(format, args)
	entry.write()
}

func (entry *Entry) write() {
	serialized, err := entry.Logger.Formatter.Format(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to obtain reader, %v\n", err)
		return
	}

	if _, err := entry.Out.Write(serialized); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write to log, %v\n", err)
	}
}

func (entry *Entry) parseMessage(format string, args []any) string {
	if len(args) == 0 {
		return format
	}

	var (
		msg    string
		fields []any
	)

	if len(format) > 0 {
		msg = fmt.Sprintf(format, args...)
	} else {
		if len(args) > 1 {
			fields = args[1:]
			args = args[:1]
		}
		msg = fmt.Sprint(args...)
	}

	if len(fields) > 0 {
		for i := 0; i < len(fields); i = i + 2 {
			if len(fields)-i < 2 {
				break
			}
			if key, ok := fields[i].(string); ok {
				entry.data[key] = fields[i+1]
			}
		}
	}
	return msg
}

type Formatter interface {
	Format(*Entry) ([]byte, error)
}

func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

type JSONFormatter struct{}

func (f *JSONFormatter) Format(entry *Entry) ([]byte, error) {
	var b *bytes.Buffer
	if entry.buf != nil {
		b = entry.buf
	} else {
		b = &bytes.Buffer{}
	}

	encoder := json.NewEncoder(b)
	if err := encoder.Encode(entry.data); err != nil {
		return nil, fmt.Errorf("failed to marshal fields to JSON, %w", err)
	}

	return b.Bytes(), nil
}

var (
	defaultBufPool = &defaultPool{
		pool: &sync.Pool{
			New: func() any {
				return new(bytes.Buffer)
			},
		},
	}
)

type defaultPool struct {
	pool *sync.Pool
}

func (p *defaultPool) Get() *bytes.Buffer {
	return p.pool.Get().(*bytes.Buffer)
}

func (p *defaultPool) Put(b *bytes.Buffer) {
	b.Reset()
	p.pool.Put(b)
}
