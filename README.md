RAOP Daemon. Implements AirTunes functionality. 

Install
-------

` $ go install github.com/maghul/go.raopd `

Dependencies
------------
	"github.com/guelfey/go.dbus"
	"github.com/maghul/go.alac"
	"github.com/maghul/go.dnssd"
	"github.com/maghul/go.slf"
	"github.com/miekg/dns"

	"github.com/stretchr/testify/assert"


License
-------
The raopd package is licensed under the LGPL with an exception that allows it to be linked statically. Please see the LICENSE file for details.

Documentation
-------------
Reference API documentation can be generated using godoc. The documentation can
alse be found at https://godoc.org/github.com/maghul/go.raopd

Usage
-----
The package defines Sources and Sinks.

A Source is an AirPlay source which will be named and published and can
be connected to from an AirPlay client.

A Sink is an interface that needs to be implemented to handle the audio
and from the AirPlay client.

For more specific needs the API exposes CreateRecordRegistrar for registering
arbitrary entries.


Examples
--------


func main() {
	keyfilename := "/tmp/airport.key"
	airplayers, err = raopd.NewSinkCollection(keyfilename)
	if err != nil {
		panic(err)
	}

	sc := make(chan os.Signal)
	signal.Notify(sc)
	go func() {
		for {
			s := <-sc
			slog.Info("Received Signal: ", s)
			if s == os.Interrupt || s == os.Kill || s == syscall.SIGTERM {
				shutdownAll()
			}
		}
	}()
	
		si := &raopd.SinkInfo{
			SupportsCoverArt: true,
			SupportsMetaData: "JSON",
			Name:             name,
			HardwareAddress:  hwaddr,
			Port:             0,
		}

		sp := &SqueezePlayer{nil, si, nil, nil, nil, "", h, name}
		sp.initPlayer()

		sp.apService, err = airplayers.Register(sp)
		if err != nil {
			return nil, err
		}

	airplayers.Close()
	