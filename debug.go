package raopd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

type iLogger interface {
	Println(d ...interface{})
}

type loggers struct {
	Debug, Info iLogger
}

var loggermap map[string]*loggers = make(map[string]*loggers)
var loggermapmutex sync.Mutex

type nullLoggerImpl struct{}

type loggerImpl func(d ...interface{})

func (li loggerImpl) Println(d ...interface{}) {
	li(d...)
}

func (nli nullLoggerImpl) Println(d ...interface{}) {

}

func getLogger(name string) *loggers {
	loggermapmutex.Lock()
	defer loggermapmutex.Unlock()

	l, ok := loggermap[name]
	if !ok {
		l = &loggers{nullLoggerImpl{}, nullLoggerImpl{}}
		loggermap[name] = l
	}
	return l
}

func getLoggerImplementation(name string, logger interface{}) (iLogger, error) {
	if logger == nil {
		return nullLoggerImpl{}, nil
	}

	if li, ok := logger.(iLogger); ok {
		var lgi loggerImpl = func(d ...interface{}) {
			b := make([]interface{}, 0, 2+len(d))
			b = append(b, name, ":")
			b = append(b, d...)
			li.Println(b...)
		}
		return lgi, nil
	}

	if iowri, ok := logger.(int); ok {
		switch iowri {
		case 1:
			logger = os.Stdout
		case 2:
			logger = os.Stderr
		}
	}

	if iowr, ok := logger.(io.Writer); ok {
		var li loggerImpl = func(d ...interface{}) {
			b := make([]interface{}, 0, 2+len(d))
			b = append(b, name, ":")
			b = append(b, d...)
			fmt.Fprintln(iowr, b...)
		}
		return li, nil
	}
	return nil, errors.New(fmt.Sprintf("Could not get a logger implementation for ", logger))
}

func setLogger(name string, info bool, logger interface{}) error {
	loggermapmutex.Lock()
	defer loggermapmutex.Unlock()

	if name == "*" {
		for n, l := range loggermap {
			li, err := getLoggerImplementation(n, logger)
			if err != nil {
				return err
			}
			l.setLogger(info, li)
		}
	} else {
		li, err := getLoggerImplementation(name, logger)
		if err != nil {
			return err
		}

		l, ok := loggermap[name]
		if !ok {
			return errors.New(fmt.Sprintf("Could not find any logger named '%s'", name))
		}
		l.setLogger(info, li)
	}
	return nil
}

func (l *loggers) setLogger(info bool, li iLogger) {
	if info {
		l.Info = li
	} else {
		l.Debug = li
	}
}

func Debug(name string, value interface{}) error {
	switch {
	case name == "sequencetrace":
		flag, _ := value.(bool)
		debugSequenceLogFlag = flag
		return nil
	case strings.HasPrefix(name, "log.info/"):
		return setLogger(name[9:], true, value)
	case strings.HasPrefix(name, "log.debug/"):
		return setLogger(name[10:], false, value)
	}
	return errors.New(fmt.Sprint("Debug name '", name, "' is unknown"))
}
