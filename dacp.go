package raopd

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
)

type dacp struct {
	id   string
	ar   string
	req  *zeroconfResolveRequest
	mrc  chan func() error // For unconnected requests
	crc  chan func() error // For connected requests
	addr *net.TCPAddr
}

var dacplog = GetLogger("raopd.dacp")

func newDacp() *dacp {
	d := &dacp{}
	d.mrc = make(chan func() error)
	d.crc = make(chan func() error, 12)
	go d.runDacp()
	return d
}

func (d *dacp) open(id string, ar string) {
	d.mrc <- func() error {
		if d.id == id && d.ar == ar {
			//			dacplog.Debug().Println( "Already resolved/resolving id=", d.id, ", ar=", d.ar)
		} else {
			d.addr = nil // Invalidate the old connection.
			dacplog.Debug().Println("DACP: open connection to id=", id, ", ar=", ar)
			d.id = id
			d.ar = ar

			if d.req != nil {
				d.req.close()
				// close connection as well.
			}

			var err error
			name := fmt.Sprintf("iTunes_Ctrl_%s", id)
			d.req, err = resolveService(name, "_dacp._tcp")
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func (d *dacp) close() {
	d.mrc <- func() error {
		dacplog.Debug().Println("Closing current DACP session.")
		d.id = ""
		d.ar = ""
		d.addr = nil
		d.req.close()
		d.req = nil
		return nil
	}
}

func (d *dacp) dacpID() string {
	return d.id
}

func (d *dacp) activeRemote() string {
	return d.ar
}

/*
beginff 	begin fast forward
beginrew 	begin rewind
mutetoggle 	toggle mute status
nextitem 	play next item in playlist
previtem 	play previous item in playlist
pause 	pause playback
playpause 	toggle between play and pause
play 	start playback
stop 	stop playback
playresume 	play after fast forward or rewind
shuffle_songs 	shuffle playlist
volumedown 	turn audio volume down
volumeup 	turn audio volume up
*/
func (d *dacp) tx(cmd string) {
	d.crc <- func() error {
		dacplog.Debug().Println("Sending Command", cmd)
		url := fmt.Sprintf("http://%s/ctrl-int/1/%s", d.addr, cmd)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "DACP: error in request", err)
		}
		req.Header.Add("Active-Remote", d.ar)
		dacplog.Debug().Println("DACP: req=", req.URL)
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			if resp.StatusCode != 200 {
				return errors.New(fmt.Sprintf("error in DACP response: '%s'", resp.Status))
			}
		}
		return err
	}
}

func (d *dacp) runDacp() {
	var err error

	for {

		switch {
		case d.req == nil:
			dacplog.Debug.Println("runDacp: event wait: d.req=nil")
			select {
			case dr := <-d.mrc:
				err = dr()
			case dr := <-d.crc:
				err = dr()
			}
		case d.req != nil:
			select {
			case dr := <-d.mrc:
				err = dr()
			case dr := <-d.crc:
				err = dr()
			case rr := <-d.req.result:
				dacplog.Debug.Println("DACP RR=", rr)
				d.addr = rr.addr
			}
		default:
			dr := <-d.mrc
			err = dr()

		}
		if err != nil {
			dacplog.Info.Println(err)
			d.addr = nil // Should we try to ask again?
		}
	}
}
