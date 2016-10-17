package raopd

import (
	"emh/logger"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
)

type dacp struct {
	sink Sink
	id   string
	ar   string
	req  *zeroconfResolveRequest
	mrc  chan func() error // For unconnected requests
	crc  chan func() error // For connected requests

	addr4         *net.TCPAddr
	addr6         *net.TCPAddr
	connectedName string
}

var dacplog = logger.GetLogger("raopd.dacp")

func newDacp(sink Sink) *dacp {
	d := &dacp{}
	d.mrc = make(chan func() error)
	d.crc = make(chan func() error, 12)
	d.sink = sink
	go d.runDacp()
	return d
}

func (d *dacp) open(id string, ar string) {
	d.mrc <- func() error {
		if d.id == id && d.ar == ar {
			//			dacplog.Debug().Println( "Already resolved/resolving id=", d.id, ", ar=", d.ar)
		} else {
			d.addr4 = nil // Invalidate the old connection.
			d.addr6 = nil // Invalidate the old connection.
			dacplog.Debug.Println("DACP: open connection to id=", id, ", ar=", ar)
			d.id = id
			d.ar = ar

			if d.req != nil {
				zeroconf().close(d.req)
				// close connection as well.
			}

			var err error
			name := fmt.Sprintf("iTunes_Ctrl_%s", id)
			d.req, err = zeroconf().resolveService(name, "_dacp._tcp")
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func (d *dacp) close() {
	d.mrc <- func() error {
		dacplog.Debug.Println("Closing current DACP session.")
		d.id = ""
		d.ar = ""
		d.addr4 = nil
		d.addr6 = nil
		zeroconf().close(d.req)
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
		dacplog.Debug.Println("Sending Command", cmd)
		url, err := d.getCommandUrl(cmd)
		if err != nil {
			return err
		}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}
		req.Header.Add("Active-Remote", d.ar)
		dacplog.Debug.Println("DACP: req=", req.URL)
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			if resp.StatusCode != 200 {
				return errors.New(fmt.Sprintf("error in DACP response: '%s'", resp.Status))
			}
		}
		return err
	}
}

func (d *dacp) getCommandUrl(cmd string) (string, error) {
	if d.addr4 != nil {
		return fmt.Sprintf("http://%s/ctrl-int/1/%s", d.addr4, cmd), nil
	} else if d.addr4 != nil {
		return fmt.Sprintf("http://%s/ctrl-int/1/%s", d.addr6, cmd), nil
	}
	return "", errors.New("Could not find an address for DACP URL")
}

func (d *dacp) setResolvedAddress(rr *zeroconfResolveReply) {
	dacplog.Debug.Println("DACP RR=", rr)
	ip := rr.addr.IP
	if ip.To4() != nil {
		dacplog.Info.Println("DACP Address was updated, old address=", d.addr4, ", new address=", rr.addr)
		if d.addr4 != rr.addr {
			d.addr4 = rr.addr
		}
	} else if ip.To16() != nil {
		dacplog.Info.Println("DACP Address was updated, old address=", d.addr6, ", new address=", rr.addr)
		if d.addr6 != rr.addr {
			d.addr6 = rr.addr
		}
	} else {
		dacplog.Info.Println("Internal error: Can not handle this IP address: ", rr.addr)
		os.Exit(-1)
	}

	if d.connectedName != rr.name {
		d.sink.Connected(rr.name)
		d.connectedName = rr.name
	}
}

func (d *dacp) isConnected() bool {
	return d.addr4 != nil || d.addr6 != nil
}

func (d *dacp) runDacp() {
	var err error

	for {

		switch {
		case d.isConnected() && d.req != nil:
			// This means we have an address so we can happily process our requests
			// if do get an update from zeroconf we just log it and change the address
			// for future requests.
			select {
			case dr := <-d.mrc:
				err = dr()
			case dr := <-d.crc:
				err = dr()
			case rr := <-d.req.result:
				d.setResolvedAddress(rr)
			}
		case d.req != nil:
			// This means we have no address and can't send any requests, but there
			// is a zeroconf resolve request pending se we will just wait for the
			// response.
			select {
			case dr := <-d.mrc:
				err = dr()
			case rr := <-d.req.result:
				d.setResolvedAddress(rr)
			}
		default:
			// This means we have no address and no request for DACP. Just wait
			// for a request to connect.
			dr := <-d.mrc
			err = dr()

		}
		if err != nil {
			dacplog.Info.Println(err)
			d.addr4 = nil // Should we try to ask again?
			d.addr6 = nil // Should we try to ask again?
		}
	}
}
