package raopd

import (
	"errors"
	"fmt"

	"github.com/maghul/go.slf"
)

func init() {
	r := slf.GetLogger("raopd")
	r.SetDescription("RAOPD Parent Logging")
	r.SetLevel(slf.Parent)
}

/*
 */
func Debug(name string, value interface{}) error {
	switch {
	case name == "sequencetrace":
		flag, _ := value.(bool)
		debugSequencer(flag)
		return nil
	case name == "volumetrace":
		flag, _ := value.(bool)
		volumetracelog = flag
		return nil
	}
	return errors.New(fmt.Sprint("Debug name '", name, "' is unknown"))
}

func getLogger(name string) *slf.Logger {
	l := slf.GetLogger(name)
	p := slf.GetLogger("raopd")
	l.SetParent(p)
	return l
}
