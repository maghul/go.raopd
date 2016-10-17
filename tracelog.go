package raopd

import (
	"fmt"
	"io"
	"os"
	"time"
)

type logentry struct {
	tm time.Time
	lf func(io.Writer)
}

// Capture packet numbers, timestamps and resends for later analysis
type tracelog struct {
	traceing bool
	wr       io.Writer
	lec      chan *logentry
	start    time.Time
}

func openPath(path string, multiple bool) (string, *os.File, error) {
	if !multiple {
		f, err := os.Create(path)
		return path, f, err
	}

	apath := path
	count := 0
	for {
		s, _ := os.Stat(apath)
		if s == nil {
			return openPath(apath, false)
		}
		count++
		apath = fmt.Sprintf("%s.%d", path, count)
	}
}

func (tl *tracelog) initTraceLog(name, suffix string, multiple bool) {
	path := fmt.Sprint("/tmp/", name, ".", suffix)
	var err error
	path, tl.wr, err = openPath(path, multiple)
	if err != nil {
		seqlog.Info.Println("Could not open trace log '", path, "'")
		return
	}
	tl.lec = make(chan *logentry, 128)
	go func() {
		tl.start = time.Now()
		tl.timestamp(tl.wr, tl.start)
		fmt.Fprintln(tl.wr, "Starting tracelog ", name, ".", suffix, " at ", tl.start)
		for tl.wr != nil {
			le := <-tl.lec
			tl.timestamp(tl.wr, le.tm)
			le.lf(tl.wr)
		}
	}()
	seqlog.Info.Println("Opened trace log '", path, "'")
	tl.traceing = true
}

func (tl *tracelog) timestamp(wr io.Writer, tm time.Time) {
	d := tm.Sub(tl.start)
	ns := d.Nanoseconds()
	ns %= 1000000000
	ms := ns / 1000000
	ns %= 1000000
	fmt.Fprintf(wr, "%5.5d.%3.3d.%6.6d : ", int(d.Seconds()), ms, ns)
}

func (tl *tracelog) log(logfunc func(io.Writer)) {
	if tl == nil || !tl.traceing {
		return
	}
	tl.lec <- &logentry{time.Now(), logfunc}
}

func (tl *tracelog) trace(d ...interface{}) {
	if tl == nil || !tl.traceing {
		return
	}
	tl.lec <- &logentry{time.Now(), func(wr io.Writer) {
		fmt.Fprintln(wr, d...)
	}}
}
