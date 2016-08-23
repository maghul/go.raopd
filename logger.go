package raopd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type LogLevel int

const (
	LogInfo   = LogLevel(0)
	LogDebug  = LogLevel(1)
	LogParent = LogLevel(1000)
)
const timestampFormat = "15:04:05"

func (ll LogLevel) String() string {
	switch ll {
	case LogInfo:
		return "INFO"
	case LogDebug:
		return "DEBUG"
	default:
		panic(fmt.Sprint("Unknown log level", int(ll)))
	}
}

type Logger struct {
	name       string
	parent     *Logger
	subloggers map[string]*Logger

	parentlevel bool
	level       LogLevel
	formatter   func(level LogLevel, name string, msg *bytes.Buffer)

	debug *loggerOutImpl
	info  *loggerOutImpl
}

type LoggerOut interface {
	Println(v ...interface{})
	Printf(fmt string, v ...interface{})
	Write(p []byte) (n int, err error)
}

var rootLogger = makeLogger("")

func init() {
	rootLogger.formatter = makeStandardFormatter(os.Stdout)
}

func GetLogger(name string) *Logger {
	if name == "" {
		return rootLogger
	}
	return rootLogger.GetLogger(name)
}

func makeStandardFormatter(w io.Writer) func(level LogLevel, name string, msg *bytes.Buffer) {
	var prev time.Time

	return func(level LogLevel, name string, msg *bytes.Buffer) {
		now := time.Now()

		nd := prev.Day() != now.Day()
		prev = now
		if nd {
			fmt.Fprintln(w, "")
			fmt.Fprintln(w, "New Day", now.Format(time.RFC1123))
			fmt.Fprintln(w, "")
		}

		prefix := fmt.Sprintf("%s.%3.3d:%s:%s: ",
			now.Format(timestampFormat), now.Nanosecond()/1000000, level, name)
		for _, l := range strings.Split(msg.String(), "\n") {
			fmt.Fprintln(w, prefix, l)
		}
	}
}

func makeLogger(name string) *Logger {
	sl := &Logger{}
	sl.name = name
	sl.parentlevel = true
	sl.level = LogInfo

	sl.formatter = nil

	sl.debug = &loggerOutImpl{LogDebug, sl}
	sl.info = &loggerOutImpl{LogInfo, sl}
	return sl
}

func (l *Logger) findLogger(name string) *Logger {
	if l.subloggers == nil {
		return nil
	}
	sl := l.subloggers[name]
	return sl
}

func (l *Logger) SetOutput(w io.Writer) {
	l.formatter = makeStandardFormatter(w)
}

func (l *Logger) findOrMakeLogger(name string) *Logger {
	sl := l.findLogger(name)
	if sl != nil {
		return sl
	}

	path := name
	if l.name != "" {
		path = fmt.Sprint(l.name, ".", name)
	}

	sl = makeLogger(path)
	sl.level = l.level
	sl.parent = l

	if l.subloggers == nil {
		l.subloggers = make(map[string]*Logger)
	}
	l.subloggers[name] = sl
	return sl
}

func (l *Logger) GetLogger(path string) *Logger {
	pl := l
	for _, name := range strings.Split(path, ".") {
		pl = pl.findOrMakeLogger(name)
	}
	return pl
}

func (l *Logger) SetLevel(lvl LogLevel) {
	if lvl == LogParent {
		l.parentlevel = true
		l.SetLevel(l.parent.level)
	} else {
		l.level = lvl
		if l.subloggers != nil {
			for _, sl := range l.subloggers {
				if sl.parentlevel {
					sl.SetLevel(lvl)
				}
			}
		}
	}
}

func (l *Logger) Info() LoggerOut {
	if l.level >= LogInfo {
		return l.info
	}
	return nilLogger
}

func (l *Logger) Debug() LoggerOut {
	if l.level >= LogDebug {
		return l.debug
	}
	return nilLogger
}

func (l *Logger) format(lvl LogLevel, msg *bytes.Buffer) {
	fl := l
	for ; fl.formatter == nil; fl = fl.parent {
	}

	fl.formatter(lvl, l.name, msg)
}

type loggerOutImpl struct {
	t LogLevel
	l *Logger
}

func (lo *loggerOutImpl) Println(v ...interface{}) {
	b := bytes.NewBufferString("")
	fmt.Fprint(b, v...)
	lo.l.format(lo.t, b)

}

func (lo *loggerOutImpl) Printf(format string, v ...interface{}) {
	b := bytes.NewBufferString("")
	fmt.Fprintf(b, format, v...)
	lo.l.format(lo.t, b)
}

func (lo *loggerOutImpl) Write(p []byte) (n int, err error) {
	return len(p), nil
}

type loggerOutNil int

const nilLogger = loggerOutNil(0)

func (lo loggerOutNil) Println(v ...interface{}) {
}

func (lo loggerOutNil) Printf(format string, v ...interface{}) {
}

func (lo loggerOutNil) Write(p []byte) (n int, err error) {
	return len(p), nil
}
