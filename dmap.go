package raopd

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

// This is the DMAP tag handler. It will read the binary encoded DMAP metadata
// and decode it.

var dmaplog = getLogger("raopd.dmap", "Digital Media Access Protocol")

type dmap struct {
	data []byte
}

func newDmap(r io.Reader) (*dmap, error) {
	initDmap()
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return &dmap{buf}, nil
}

func (d *dmap) SetData(data []byte) {
	d.data = data
}

func (d *dmap) String(format string) string {
	format = strings.ToLower(format)
	if format == "xml" || format == "json" {
		b := bytes.NewBufferString("")
		d.Write(b, format)
		return b.String()
	} else {
		return ""
	}
}

func (d *dmap) Write(w io.Writer, format string) {
	json := checkDmapFormat(format)
	bw := bufio.NewWriter(w)
	d.WriteX(bw, json)
	bw.Flush()
}

func checkDmapFormat(format string) bool {
	switch strings.ToLower(format) {
	case "xml":
		return false
	case "json":
		return true
	default:
		dmaplog.Info.Println("Can not print format '%s'", format)
		os.Exit(-1)
	}
	return false
}

func (d *dmap) WriteX(w *bufio.Writer, json bool) {
	indent := 0
	if json {
		indent = 4
		w.WriteString("{\r\n")
	} else {
		w.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\r\n")
	}

	dmapWriteEntry(w, json, d.data, indent)

	if json {
		w.WriteString("}\r\n")
	} else {
		w.WriteString("\r\n")
	}

}

func writeSpaces(w *bufio.Writer, spaces int) {
	for ; spaces > 0; spaces-- {
		w.WriteRune(' ')
	}
}

func dmapIntToUint64(d []byte) uint64 {
	// Is there a variable length decoder built-in?
	v := uint64(0)
	for ii := 0; ii < len(d); ii++ {
		v = v<<8 | uint64(d[ii])
	}
	return v
}

func dmapPrintInt(w *bufio.Writer, json bool, data []byte, indent int, length int) []byte {
	fmt.Fprint(w, dmapIntToUint64(data[0:length]))
	return data[length:]
}

func dmapPrintUint(w *bufio.Writer, json bool, data []byte, indent int, length int) []byte {
	fmt.Fprint(w, int64(dmapIntToUint64(data[0:length])))
	return data[length:]
}

func dmapPrintData(w *bufio.Writer, json bool, data []byte, indent int, length int) []byte {
	s := fmt.Sprint(data[0:length])
	s = strings.Replace(s, " ", ", ", -1)

	if json {
		fmt.Fprint(w, s)
	} else {
		fmt.Fprint(w, "<!CDATA<", s, ">>")
	}
	return data[length:]
}

func dmapPrintStr(w *bufio.Writer, json bool, data []byte, indent int, length int) []byte {
	if json {
		w.WriteRune('"')
		w.WriteString(string(data[0:length]))
		w.WriteRune('"')
	} else {
		w.WriteString(string(data[0:length]))
	}
	return data[length:]
}

func dmapPrintDate(w *bufio.Writer, json bool, data []byte, indent int, length int) []byte {
	dl := dmapIntToUint64(data[0:length])
	dr := time.Duration(dl) * time.Second
	t := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	t = t.Add(dr)
	w.WriteRune('"')
	w.WriteString(t.String())
	w.WriteRune('"')
	return data[length:]
}

func dmapPrintVers(w *bufio.Writer, json bool, data []byte, indent int, length int) []byte {
	w.WriteString("***VERS***")
	return data[length:]
}

func dmapPrintDict(w *bufio.Writer, json bool, data []byte, indent int, length int) []byte {
	var prefix, infix, postfix, lastfix string

	if json {
		prefix = "{\r\n"
		infix = ",\r\n"
		postfix = "}\r\n"
		lastfix = "\r\n"
	} else {
		prefix = "\r\n"
		infix = "\r\n"
		postfix = "\r\n"
		lastfix = ""
	}

	sep := prefix
	dsource := data[0:length]
	indent += 4

	for len(dsource) > 0 {
		w.WriteString(sep)
		dsource = dmapWriteEntry(w, json, dsource, indent)
		sep = infix
	}
	if sep != infix {
		w.WriteString(prefix)
	} else {
		w.WriteString(lastfix)
	}
	writeSpaces(w, indent-4)
	w.WriteString(postfix)
	return data[length:]
}

func dmapWriteEntry(w *bufio.Writer, json bool, data []byte, indent int) []byte {
	tag := string(data[0:4])
	len := int(binary.BigEndian.Uint32(data[4:]))

	writeSpaces(w, indent)

	var nd []byte

	entry, ok := dmapEntryMap[tag]
	entryname := tag
	if ok {
		entryname = entry.name
	}
	if json {
		w.WriteRune('"')
		w.WriteString(entryname)
		w.WriteString("\": ")
	} else {
		w.WriteRune('<')
		w.WriteString(entryname)
		w.WriteString(">")
	}

	if !ok {
		dmaplog.Info.Println("DMAP tag '", tag, "' is not known")
		dmapPrintData(w, json, data[8:8+len], indent, len)
		nd = data[8+len:]
	} else {
		nd = entry.printer(w, json, data[8:], indent, len)
	}

	if !json {
		w.WriteString("</")
		w.WriteString(entryname)
		w.WriteString(">")
	}
	return nd
}

type dmapEntry struct {
	tag     string
	printer func(w *bufio.Writer, json bool, data []byte, indent int, length int) []byte
	name    string
}

var dmapEntryMap map[string]dmapEntry

func initDmap() {
	if dmapEntryMap != nil {
		return
	}

	dmapEntryMap = map[string]dmapEntry{

		"abal":    dmapEntry{"abal", dmapPrintDict, "daap.browsealbumlisting"},
		"abar":    dmapEntry{"abar", dmapPrintDict, "daap.browseartistlisting"},
		"abcp":    dmapEntry{"abcp", dmapPrintDict, "daap.browsecomposerlisting"},
		"abgn":    dmapEntry{"abgn", dmapPrintDict, "daap.browsegenrelisting"},
		"abpl":    dmapEntry{"abpl", dmapPrintUint, "daap.baseplaylist"},
		"abro":    dmapEntry{"abro", dmapPrintDict, "daap.databasebrowse"},
		"adbs":    dmapEntry{"adbs", dmapPrintDict, "daap.databasesongs"},
		"aeAD":    dmapEntry{"aeAD", dmapPrintDict, "com.apple.itunes.adam-ids-array"},
		"aeAI":    dmapEntry{"aeAI", dmapPrintUint, "com.apple.itunes.itms-artistid"},
		"aeAK":    dmapEntry{"aeAK", dmapPrintStr, "unknown.aeAK"},
		"aeCD":    dmapEntry{"aeCD", dmapPrintData, "com.apple.itunes.flat-chapter-data"},
		"aeCd":    dmapEntry{"aeCd", dmapPrintUint, "com.apple.itunes.cloud-id"},
		"aeCF":    dmapEntry{"aeCF", dmapPrintUint, "com.apple.itunes.cloud-flavor-id"},
		"aeCI":    dmapEntry{"aeCI", dmapPrintUint, "com.apple.itunes.itms-composerid"},
		"aeCK":    dmapEntry{"aeCK", dmapPrintUint, "com.apple.itunes.cloud-library-kind"},
		"aeCM":    dmapEntry{"aeCM", dmapPrintUint, "com.apple.itunes.cloud-match-type"},
		"aecp":    dmapEntry{"aecp", dmapPrintStr, "com.apple.itunes.collection-description"},
		"aeCR":    dmapEntry{"aeCR", dmapPrintStr, "com.apple.itunes.content-rating"},
		"aeCs":    dmapEntry{"aeCs", dmapPrintUint, "com.apple.itunes.artworkchecksum"},
		"aeCS":    dmapEntry{"aeCS", dmapPrintUint, "com.apple.itunes.artworkchecksum"},
		"aeCU":    dmapEntry{"aeCU", dmapPrintUint, "com.apple.itunes.cloud-user-id"},
		"aeDL":    dmapEntry{"aeDL", dmapPrintUint, "com.apple.itunes.drm-downloader-user-id"},
		"aeDP":    dmapEntry{"aeDP", dmapPrintUint, "com.apple.itunes.drm-platform-id"},
		"aeDR":    dmapEntry{"aeDR", dmapPrintUint, "com.apple.itunes.drm-user-id"},
		"aeDV":    dmapEntry{"aeDV", dmapPrintUint, "com.apple.itunes.drm-versions"},
		"aeEN":    dmapEntry{"aeEN", dmapPrintStr, "com.apple.itunes.episode-num-str"},
		"aeES":    dmapEntry{"aeES", dmapPrintUint, "com.apple.itunes.episode-sort"},
		"aeFA":    dmapEntry{"aeFA", dmapPrintUint, "com.apple.itunes.drm-family-id"},
		"aeFP":    dmapEntry{"aeFP", dmapPrintUint, "com.apple.itunes.unknown-FP"},
		"aeFR":    dmapEntry{"aeFR", dmapPrintUint, "com.apple.itunes.unknown-FR"},
		"aeGD":    dmapEntry{"aeGD", dmapPrintUint, "com.apple.itunes.gapless-enc-dr"},
		"aeGE":    dmapEntry{"aeGE", dmapPrintUint, "com.apple.itunes.gapless-enc-del"},
		"aeGH":    dmapEntry{"aeGH", dmapPrintUint, "com.apple.itunes.gapless-heur"},
		"aeGI":    dmapEntry{"aeGI", dmapPrintUint, "com.apple.itunes.itms-genreid"},
		"aeGR":    dmapEntry{"aeGR", dmapPrintUint, "com.apple.itunes.gapless-resy"},
		"aeGs":    dmapEntry{"aeGs", dmapPrintUint, "com.apple.itunes.can-be-genius-seed"},
		"aeGU":    dmapEntry{"aeGU", dmapPrintUint, "com.apple.itunes.gapless-dur"},
		"aeHC":    dmapEntry{"aeHC", dmapPrintUint, "com.apple.itunes.has-chapter-data"},
		"aeHD":    dmapEntry{"aeHD", dmapPrintUint, "com.apple.itunes.is-hd-video"},
		"aeHV":    dmapEntry{"aeHV", dmapPrintUint, "com.apple.itunes.has-video"},
		"aeIM":    dmapEntry{"aeIM", dmapPrintUint, "com.apple.itunes.unknown-IM"},
		"aeK1":    dmapEntry{"aeK1", dmapPrintUint, "com.apple.itunes.drm-key1-id"},
		"aeK2":    dmapEntry{"aeK2", dmapPrintUint, "com.apple.itunes.drm-key2-id"},
		"aels":    dmapEntry{"aels", dmapPrintUint, "com.apple.itunes.liked-state"},
		"aeMC":    dmapEntry{"aeMC", dmapPrintUint, "com.apple.itunes.playlist-contains-media-type-count"},
		"aemi":    dmapEntry{"aemi", dmapPrintDict, "com.apple.itunes.media-kind-listing-item"},
		"aeMk":    dmapEntry{"aeMk", dmapPrintUint, "com.apple.itunes.extended-media-kind"},
		"aeMK":    dmapEntry{"aeMK", dmapPrintUint, "com.apple.itunes.mediakind"},
		"aeml":    dmapEntry{"aeml", dmapPrintDict, "com.apple.itunes.media-kind-listing"},
		"aeMQ":    dmapEntry{"aeMQ", dmapPrintUint, "com.apple.itunes.unknown-MQ"},
		"aeMX":    dmapEntry{"aeMX", dmapPrintStr, "com.apple.itunes.movie-info-xml"},
		"aeND":    dmapEntry{"aeND", dmapPrintUint, "com.apple.itunes.non-drm-user-id"},
		"aeNN":    dmapEntry{"aeNN", dmapPrintStr, "com.apple.itunes.network-name"},
		"aeNV":    dmapEntry{"aeNV", dmapPrintUint, "com.apple.itunes.norm-volume"},
		"aePC":    dmapEntry{"aePC", dmapPrintUint, "com.apple.itunes.is-podcast"},
		"aePI":    dmapEntry{"aePI", dmapPrintUint, "com.apple.itunes.itms-playlistid"},
		"aePP":    dmapEntry{"aePP", dmapPrintUint, "com.apple.itunes.is-podcast-playlist"},
		"aePS":    dmapEntry{"aePS", dmapPrintUint, "com.apple.itunes.special-playlist"},
		"aeRD":    dmapEntry{"aeRD", dmapPrintUint, "com.apple.itunes.rental-duration"},
		"aeRf":    dmapEntry{"aeRf", dmapPrintUint, "com.apple.itunes.is-featured"},
		"aeRM":    dmapEntry{"aeRM", dmapPrintUint, "com.apple.itunes.unknown-RM"},
		"aeRP":    dmapEntry{"aeRP", dmapPrintUint, "com.apple.itunes.rental-pb-start"},
		"aeRS":    dmapEntry{"aeRS", dmapPrintUint, "com.apple.itunes.rental-start"},
		"aeRU":    dmapEntry{"aeRU", dmapPrintUint, "com.apple.itunes.rental-pb-duration"},
		"aeSE":    dmapEntry{"aeSE", dmapPrintUint, "com.apple.itunes.store-pers-id"},
		"aeSF":    dmapEntry{"aeSF", dmapPrintUint, "com.apple.itunes.itms-storefrontid"},
		"aeSG":    dmapEntry{"aeSG", dmapPrintUint, "com.apple.itunes.saved-genius"},
		"aeSI":    dmapEntry{"aeSI", dmapPrintUint, "com.apple.itunes.itms-songid"},
		"aeSL":    dmapEntry{"aeSL", dmapPrintUint, "com.apple.itunes.unknown-SL"},
		"aeSN":    dmapEntry{"aeSN", dmapPrintStr, "com.apple.itunes.series-name"},
		"aeSP":    dmapEntry{"aeSP", dmapPrintUint, "com.apple.itunes.smart-playlist"},
		"aeSR":    dmapEntry{"aeSR", dmapPrintUint, "com.apple.itunes.unknown-SR"},
		"aeSU":    dmapEntry{"aeSU", dmapPrintUint, "com.apple.itunes.season-num"},
		"aeSV":    dmapEntry{"aeSV", dmapPrintVers, "com.apple.itunes.music-sharing-version"},
		"aeSX":    dmapEntry{"aeSX", dmapPrintUint, "com.apple.itunes.unknown-SX"},
		"aeTr":    dmapEntry{"aeTr", dmapPrintUint, "com.apple.itunes.unknown-Tr"},
		"aeXD":    dmapEntry{"aeXD", dmapPrintStr, "com.apple.itunes.xid"},
		"agac":    dmapEntry{"agac", dmapPrintUint, "daap.groupalbumcount"},
		"agal":    dmapEntry{"agal", dmapPrintDict, "com.apple.itunes.unknown-al"},
		"agar":    dmapEntry{"agar", dmapPrintDict, "unknown.agar"},
		"agma":    dmapEntry{"agma", dmapPrintUint, "daap.groupmatchedqueryalbumcount"},
		"agmi":    dmapEntry{"agmi", dmapPrintUint, "daap.groupmatchedqueryitemcount"},
		"agrp":    dmapEntry{"agrp", dmapPrintStr, "daap.songgrouping"},
		"ajal":    dmapEntry{"ajal", dmapPrintUint, "com.apple.itunes.store.album-liked-state"},
		"ajca":    dmapEntry{"ajca", dmapPrintUint, "com.apple.itunes.store.show-composer-as-artist"},
		"ajcA":    dmapEntry{"ajcA", dmapPrintUint, "com.apple.itunes.store.show-composer-as-artist"},
		"aply":    dmapEntry{"aply", dmapPrintDict, "daap.databaseplaylists"},
		"aprm":    dmapEntry{"aprm", dmapPrintUint, "daap.playlistrepeatmode"},
		"apro":    dmapEntry{"apro", dmapPrintVers, "daap.protocolversion"},
		"apsm":    dmapEntry{"apsm", dmapPrintUint, "daap.playlistshufflemode"},
		"apso":    dmapEntry{"apso", dmapPrintDict, "daap.playlistsongs"},
		"arif":    dmapEntry{"arif", dmapPrintDict, "daap.resolveinfo"},
		"arsv":    dmapEntry{"arsv", dmapPrintDict, "daap.resolve"},
		"asaa":    dmapEntry{"asaa", dmapPrintStr, "daap.songalbumartist"},
		"asac":    dmapEntry{"asac", dmapPrintUint, "daap.songartworkcount"},
		"asai":    dmapEntry{"asai", dmapPrintUint, "daap.songalbumid"},
		"asal":    dmapEntry{"asal", dmapPrintStr, "daap.songalbum"},
		"asar":    dmapEntry{"asar", dmapPrintStr, "daap.songartist"},
		"asas":    dmapEntry{"asas", dmapPrintUint, "daap.songalbumuserratingstatus"},
		"asbk":    dmapEntry{"asbk", dmapPrintUint, "daap.bookmarkable"},
		"asbo":    dmapEntry{"asbo", dmapPrintUint, "daap.songbookmark"},
		"asbr":    dmapEntry{"asbr", dmapPrintUint, "daap.songbitrate"},
		"asbt":    dmapEntry{"asbt", dmapPrintUint, "daap.songbeatsperminute"},
		"ascd":    dmapEntry{"ascd", dmapPrintUint, "daap.songcodectype"},
		"ascm":    dmapEntry{"ascm", dmapPrintStr, "daap.songcomment"},
		"ascn":    dmapEntry{"ascn", dmapPrintStr, "daap.songcontentdescription"},
		"asco":    dmapEntry{"asco", dmapPrintUint, "daap.songcompilation"},
		"ascp":    dmapEntry{"ascp", dmapPrintStr, "daap.songcomposer"},
		"ascr":    dmapEntry{"ascr", dmapPrintUint, "daap.songcontentrating"},
		"ascs":    dmapEntry{"ascs", dmapPrintUint, "daap.songcodecsubtype"},
		"asct":    dmapEntry{"asct", dmapPrintStr, "daap.songcategory"},
		"asda":    dmapEntry{"asda", dmapPrintDate, "daap.songdateadded"},
		"asdb":    dmapEntry{"asdb", dmapPrintUint, "daap.songdisabled"},
		"asdc":    dmapEntry{"asdc", dmapPrintUint, "daap.songdisccount"},
		"asdk":    dmapEntry{"asdk", dmapPrintUint, "daap.songdatakind"},
		"asdm":    dmapEntry{"asdm", dmapPrintDate, "daap.songdatemodified"},
		"asdn":    dmapEntry{"asdn", dmapPrintUint, "daap.songdiscnumber"},
		"asdp":    dmapEntry{"asdp", dmapPrintDate, "daap.songdatepurchased"},
		"asdr":    dmapEntry{"asdr", dmapPrintDate, "daap.songdatereleased"},
		"asdt":    dmapEntry{"asdt", dmapPrintStr, "daap.songdescription"},
		"ased":    dmapEntry{"ased", dmapPrintUint, "daap.songextradata"},
		"aseq":    dmapEntry{"aseq", dmapPrintStr, "daap.songeqpreset"},
		"ases":    dmapEntry{"ases", dmapPrintUint, "daap.songexcludefromshuffle"},
		"asfm":    dmapEntry{"asfm", dmapPrintStr, "daap.songformat"},
		"asgn":    dmapEntry{"asgn", dmapPrintStr, "daap.songgenre"},
		"asgp":    dmapEntry{"asgp", dmapPrintUint, "daap.songgapless"},
		"asgr":    dmapEntry{"asgr", dmapPrintUint, "daap.supportsgroups"},
		"ashp":    dmapEntry{"ashp", dmapPrintUint, "daap.songhasbeenplayed"},
		"askd":    dmapEntry{"askd", dmapPrintDate, "daap.songlastskipdate"},
		"askp":    dmapEntry{"askp", dmapPrintUint, "daap.songuserskipcount"},
		"asky":    dmapEntry{"asky", dmapPrintStr, "daap.songkeywords"},
		"aslc":    dmapEntry{"aslc", dmapPrintStr, "daap.songlongcontentdescription"},
		"aslr":    dmapEntry{"aslr", dmapPrintUint, "daap.songalbumuserrating"},
		"asls":    dmapEntry{"asls", dmapPrintUint, "daap.songlongsize"},
		"aspc":    dmapEntry{"aspc", dmapPrintUint, "daap.songuserplaycount"},
		"aspl":    dmapEntry{"aspl", dmapPrintDate, "daap.songdateplayed"},
		"aspu":    dmapEntry{"aspu", dmapPrintStr, "daap.songpodcasturl"},
		"asri":    dmapEntry{"asri", dmapPrintUint, "daap.songartistid"},
		"asrs":    dmapEntry{"asrs", dmapPrintUint, "daap.songuserratingstatus"},
		"asrv":    dmapEntry{"asrv", dmapPrintInt, "daap.songrelativevolume"},
		"assa":    dmapEntry{"assa", dmapPrintStr, "daap.sortartist"},
		"assc":    dmapEntry{"assc", dmapPrintStr, "daap.sortcomposer"},
		"asse":    dmapEntry{"asse", dmapPrintUint, "com.apple.itunes.unknown-se"},
		"assl":    dmapEntry{"assl", dmapPrintStr, "daap.sortalbumartist"},
		"assn":    dmapEntry{"assn", dmapPrintStr, "daap.sortname"},
		"assp":    dmapEntry{"assp", dmapPrintUint, "daap.songstoptime"},
		"assr":    dmapEntry{"assr", dmapPrintUint, "daap.songsamplerate"},
		"asss":    dmapEntry{"asss", dmapPrintStr, "daap.sortseriesname"},
		"asst":    dmapEntry{"asst", dmapPrintUint, "daap.songstarttime"},
		"assu":    dmapEntry{"assu", dmapPrintStr, "daap.sortalbum"},
		"assz":    dmapEntry{"assz", dmapPrintUint, "daap.songsize"},
		"astc":    dmapEntry{"astc", dmapPrintUint, "daap.songtrackcount"},
		"astm":    dmapEntry{"astm", dmapPrintUint, "daap.songtime"},
		"astn":    dmapEntry{"astn", dmapPrintUint, "daap.songtracknumber"},
		"asul":    dmapEntry{"asul", dmapPrintStr, "daap.songdataurl"},
		"asur":    dmapEntry{"asur", dmapPrintUint, "daap.songuserrating"},
		"asvc":    dmapEntry{"asvc", dmapPrintUint, "daap.songprimaryvideocodec"},
		"asyr":    dmapEntry{"asyr", dmapPrintUint, "daap.songyear"},
		"ated":    dmapEntry{"ated", dmapPrintUint, "daap.supportsextradata"},
		"avdb":    dmapEntry{"avdb", dmapPrintDict, "daap.serverdatabases"},
		"caar":    dmapEntry{"caar", dmapPrintUint, "dacp.availablerepeatstates"},
		"caas":    dmapEntry{"caas", dmapPrintUint, "dacp.availableshufflestates"},
		"caci":    dmapEntry{"caci", dmapPrintDict, "dacp.controlint"},
		"cads":    dmapEntry{"cads", dmapPrintUint, "unknown-ds.cads"},
		"cafe":    dmapEntry{"cafe", dmapPrintUint, "dacp.fullscreenenabled"},
		"cafs":    dmapEntry{"cafs", dmapPrintUint, "dacp.fullscreen"},
		"caia":    dmapEntry{"caia", dmapPrintUint, "dacp.isactive"},
		"caip":    dmapEntry{"caip", dmapPrintUint, "com.apple.itunes.unknown-ip"},
		"caiv":    dmapEntry{"caiv", dmapPrintUint, "com.apple.itunes.unknown-iv"},
		"caks":    dmapEntry{"caks", dmapPrintUint, "unknown.ss.caks"},
		"cana":    dmapEntry{"cana", dmapPrintStr, "dacp.nowplayingartist"},
		"cang":    dmapEntry{"cang", dmapPrintStr, "dacp.nowplayinggenre"},
		"canl":    dmapEntry{"canl", dmapPrintStr, "dacp.nowplayingalbum"},
		"cann":    dmapEntry{"cann", dmapPrintStr, "dacp.nowplayingname"},
		"canp":    dmapEntry{"canp", dmapPrintData, "dacp.nowplayingids"},
		"cant":    dmapEntry{"cant", dmapPrintUint, "dacp.nowplayingtime"},
		"caov":    dmapEntry{"caov", dmapPrintUint, "unknown.ov.caov"},
		"capr":    dmapEntry{"capr", dmapPrintVers, "dacp.protocolversion"},
		"caps":    dmapEntry{"caps", dmapPrintUint, "dacp.playerstate"},
		"carp":    dmapEntry{"carp", dmapPrintUint, "dacp.repeatstate"},
		"casa":    dmapEntry{"casa", dmapPrintUint, "com.apple.itunes.unknown-sa"},
		"casc":    dmapEntry{"casc", dmapPrintUint, "unknown.ss.casc"},
		"cash":    dmapEntry{"cash", dmapPrintUint, "dacp.shufflestate"},
		"casp":    dmapEntry{"casp", dmapPrintDict, "dacp.speakers"},
		"cass":    dmapEntry{"cass", dmapPrintUint, "unknown.ss.cass"},
		"cast":    dmapEntry{"cast", dmapPrintUint, "dacp.songtime"},
		"casu":    dmapEntry{"casu", dmapPrintUint, "com.apple.itunes.unknown-su"},
		"cavc":    dmapEntry{"cavc", dmapPrintUint, "dacp.volumecontrollable"},
		"cavd":    dmapEntry{"cavd", dmapPrintUint, "com.apple.itunes.unknown-vd"},
		"cave":    dmapEntry{"cave", dmapPrintUint, "dacp.visualizerenabled"},
		"cavs":    dmapEntry{"cavs", dmapPrintUint, "dacp.visualizer"},
		"ceGS":    dmapEntry{"ceGS", dmapPrintUint, "com.apple.itunes.genius-selectable"},
		"ceJC":    dmapEntry{"ceJC", dmapPrintUint, "com.apple.itunes.jukebox-client-vote"},
		"ceJI":    dmapEntry{"ceJI", dmapPrintUint, "com.apple.itunes.jukebox-current"},
		"ceJS":    dmapEntry{"ceJS", dmapPrintUint, "com.apple.itunes.jukebox-score"},
		"ceJV":    dmapEntry{"ceJV", dmapPrintUint, "com.apple.itunes.jukebox-vote"},
		"ceQa":    dmapEntry{"ceQa", dmapPrintStr, "com.apple.itunes.playqueue-album"},
		"ceQg":    dmapEntry{"ceQg", dmapPrintStr, "com.apple.itunes.playqueue-genre"},
		"ceQh":    dmapEntry{"ceQh", dmapPrintStr, "unknown.ceQh"},
		"ceQi":    dmapEntry{"ceQi", dmapPrintUint, "unknown.ceQi"},
		"ceQI":    dmapEntry{"ceQI", dmapPrintUint, "unknown.ceQI"},
		"ceQk":    dmapEntry{"ceQk", dmapPrintStr, "unknown.ceQk"},
		"ceQl":    dmapEntry{"ceQl", dmapPrintStr, "unknown.ceQl"},
		"ceQm":    dmapEntry{"ceQm", dmapPrintUint, "unknown.ceQm"},
		"ceQn":    dmapEntry{"ceQn", dmapPrintStr, "com.apple.itunes.playqueue-name"},
		"ceQR":    dmapEntry{"ceQR", dmapPrintDict, "com.apple.itunes.playqueue-contents-response"},
		"ceQr":    dmapEntry{"ceQr", dmapPrintStr, "com.apple.itunes.playqueue-artist"},
		"ceQS":    dmapEntry{"ceQS", dmapPrintDict, "com.apple.itunes.playqueue-content-unknown"},
		"ceQs":    dmapEntry{"ceQs", dmapPrintStr, "com.apple.itunes.playqueue-id"},
		"ceQu":    dmapEntry{"ceQu", dmapPrintUint, "com.apple.itunes.unknown-Qu"},
		"ceSG":    dmapEntry{"ceSG", dmapPrintUint, "com.apple.itunes.saved-genius"},
		"ceSX":    dmapEntry{"ceSX", dmapPrintUint, "unknown.sx.ceSX"},
		"ceVO":    dmapEntry{"ceVO", dmapPrintUint, "com.apple.itunes.unknown-voting"},
		"cmgt":    dmapEntry{"cmgt", dmapPrintDict, "dmcp.getpropertyresponse"},
		"cmik":    dmapEntry{"cmik", dmapPrintUint, "unknown-ik.cmik"},
		"cmmk":    dmapEntry{"cmmk", dmapPrintUint, "dmcp.mediakind"},
		"cmnm":    dmapEntry{"cmnm", dmapPrintStr, "unknown-nm.cmnm"},
		"cmpa":    dmapEntry{"cmpa", dmapPrintDict, "unknown.pa.cmpa"},
		"cmpg":    dmapEntry{"cmpg", dmapPrintData, "com.apple.itunes.unknown-pg"},
		"cmpr":    dmapEntry{"cmpr", dmapPrintVers, "dmcp.protocolversion"},
		"cmrl":    dmapEntry{"cmrl", dmapPrintUint, "unknown.rl.cmrl"},
		"cmsp":    dmapEntry{"cmsp", dmapPrintUint, "unknown-sp.cmsp"},
		"cmsr":    dmapEntry{"cmsr", dmapPrintUint, "dmcp.serverrevision"},
		"cmst":    dmapEntry{"cmst", dmapPrintDict, "dmcp.playstatus"},
		"cmsv":    dmapEntry{"cmsv", dmapPrintUint, "unknown.sv.cmsv"},
		"cmty":    dmapEntry{"cmty", dmapPrintStr, "unknown-ty.cmty"},
		"cmvo":    dmapEntry{"cmvo", dmapPrintUint, "dmcp.volume"},
		"____":    dmapEntry{"____", dmapPrintUint, "com.apple.itunes.req-fplay"},
		"f\215ch": dmapEntry{"f\215ch", dmapPrintUint, "dmap.haschildcontainers"},
		"ipsa":    dmapEntry{"ipsa", dmapPrintDict, "dpap.iphotoslideshowadvancedoptions"},
		"ipsl":    dmapEntry{"ipsl", dmapPrintDict, "dpap.iphotoslideshowoptions"},
		"mbcl":    dmapEntry{"mbcl", dmapPrintDict, "dmap.bag"},
		"mccr":    dmapEntry{"mccr", dmapPrintDict, "dmap.contentcodesresponse"},
		"mcna":    dmapEntry{"mcna", dmapPrintStr, "dmap.contentcodesname"},
		"mcnm":    dmapEntry{"mcnm", dmapPrintUint, "dmap.contentcodesnumber"},
		"mcon":    dmapEntry{"mcon", dmapPrintDict, "dmap.container"},
		"mctc":    dmapEntry{"mctc", dmapPrintUint, "dmap.containercount"},
		"mcti":    dmapEntry{"mcti", dmapPrintUint, "dmap.containeritemid"},
		"mcty":    dmapEntry{"mcty", dmapPrintUint, "dmap.contentcodestype"},
		"mdbk":    dmapEntry{"mdbk", dmapPrintUint, "dmap.databasekind"},
		"mdcl":    dmapEntry{"mdcl", dmapPrintDict, "dmap.dictionary"},
		"mdst":    dmapEntry{"mdst", dmapPrintUint, "dmap.downloadstatus"},
		"meds":    dmapEntry{"meds", dmapPrintUint, "dmap.editcommandssupported"},
		"meia":    dmapEntry{"meia", dmapPrintUint, "dmap.itemdateadded"},
		"meip":    dmapEntry{"meip", dmapPrintUint, "dmap.itemdateplayed"},
		"mext":    dmapEntry{"mext", dmapPrintUint, "dmap.objectextradata"},
		"miid":    dmapEntry{"miid", dmapPrintUint, "dmap.itemid"},
		"mikd":    dmapEntry{"mikd", dmapPrintUint, "dmap.itemkind"},
		"mimc":    dmapEntry{"mimc", dmapPrintUint, "dmap.itemcount"},
		"minm":    dmapEntry{"minm", dmapPrintStr, "dmap.itemname"},
		"mlcl":    dmapEntry{"mlcl", dmapPrintDict, "dmap.listing"},
		"mlid":    dmapEntry{"mlid", dmapPrintUint, "dmap.sessionid"},
		"mlit":    dmapEntry{"mlit", dmapPrintDict, "dmap.listingitem"},
		"mlog":    dmapEntry{"mlog", dmapPrintDict, "dmap.loginresponse"},
		"mpco":    dmapEntry{"mpco", dmapPrintUint, "dmap.parentcontainerid"},
		"mper":    dmapEntry{"mper", dmapPrintUint, "dmap.persistentid"},
		"mpro":    dmapEntry{"mpro", dmapPrintVers, "dmap.protocolversion"},
		"mrco":    dmapEntry{"mrco", dmapPrintUint, "dmap.returnedcount"},
		"mrpr":    dmapEntry{"mrpr", dmapPrintUint, "dmap.remotepersistentid"},
		"msal":    dmapEntry{"msal", dmapPrintUint, "dmap.supportsautologout"},
		"msas":    dmapEntry{"msas", dmapPrintUint, "dmap.authenticationschemes"},
		"msau":    dmapEntry{"msau", dmapPrintUint, "dmap.authenticationmethod"},
		"msbr":    dmapEntry{"msbr", dmapPrintUint, "dmap.supportsbrowse"},
		"mscu":    dmapEntry{"mscu", dmapPrintUint, "unknown-cu.mscu"},
		"msdc":    dmapEntry{"msdc", dmapPrintUint, "dmap.databasescount"},
		"msed":    dmapEntry{"msed", dmapPrintUint, "com.apple.itunes.unknown-ed"},
		"msex":    dmapEntry{"msex", dmapPrintUint, "dmap.supportsextensions"},
		"mshc":    dmapEntry{"mshc", dmapPrintUint, "dmap.sortingheaderchar"},
		"mshi":    dmapEntry{"mshi", dmapPrintUint, "dmap.sortingheaderindex"},
		"mshl":    dmapEntry{"mshl", dmapPrintDict, "dmap.sortingheaderlisting"},
		"mshn":    dmapEntry{"mshn", dmapPrintUint, "dmap.sortingheadernumber"},
		"msix":    dmapEntry{"msix", dmapPrintUint, "dmap.supportsindex"},
		"mslr":    dmapEntry{"mslr", dmapPrintUint, "dmap.loginrequired"},
		"msma":    dmapEntry{"msma", dmapPrintUint, "dmap.machineaddress"},
		"msml":    dmapEntry{"msml", dmapPrintDict, "com.apple.itunes.unknown-ml"},
		"mspi":    dmapEntry{"mspi", dmapPrintUint, "dmap.supportspersistentids"},
		"msqy":    dmapEntry{"msqy", dmapPrintUint, "dmap.supportsquery"},
		"msrs":    dmapEntry{"msrs", dmapPrintUint, "dmap.supportsresolve"},
		"msrv":    dmapEntry{"msrv", dmapPrintDict, "dmap.serverinforesponse"},
		"mstc":    dmapEntry{"mstc", dmapPrintDate, "dmap.utctime"},
		"mstm":    dmapEntry{"mstm", dmapPrintUint, "dmap.timeoutinterval"},
		"msto":    dmapEntry{"msto", dmapPrintInt, "dmap.utcoffset"},
		"msts":    dmapEntry{"msts", dmapPrintStr, "dmap.statusstring"},
		"mstt":    dmapEntry{"mstt", dmapPrintUint, "dmap.status"},
		"msup":    dmapEntry{"msup", dmapPrintUint, "dmap.supportsupdate"},
		"mtco":    dmapEntry{"mtco", dmapPrintUint, "dmap.specifiedtotalcount"},
		"mudl":    dmapEntry{"mudl", dmapPrintDict, "dmap.deletedidlisting"},
		"mupd":    dmapEntry{"mupd", dmapPrintDict, "dmap.updateresponse"},
		"musr":    dmapEntry{"musr", dmapPrintUint, "dmap.serverrevision"},
		"muty":    dmapEntry{"muty", dmapPrintUint, "dmap.updatetype"},
		"pasp":    dmapEntry{"pasp", dmapPrintStr, "dpap.aspectratio"},
		"pcmt":    dmapEntry{"pcmt", dmapPrintStr, "dpap.imagecomments"},
		"peak":    dmapEntry{"peak", dmapPrintUint, "com.apple.itunes.photos.album-kind"},
		"peed":    dmapEntry{"peed", dmapPrintDate, "com.apple.itunes.photos.exposure-date"},
		"pefc":    dmapEntry{"pefc", dmapPrintDict, "com.apple.itunes.photos.faces"},
		"peki":    dmapEntry{"peki", dmapPrintUint, "com.apple.itunes.photos.key-image-id"},
		"pekm":    dmapEntry{"pekm", dmapPrintDict, "com.apple.itunes.photos.key-image"},
		"pemd":    dmapEntry{"pemd", dmapPrintDate, "com.apple.itunes.photos.modification-date"},
		"pfai":    dmapEntry{"pfai", dmapPrintDict, "dpap.failureids"},
		"pfdt":    dmapEntry{"pfdt", dmapPrintDict, "dpap.filedata"},
		"pfmt":    dmapEntry{"pfmt", dmapPrintStr, "dpap.imageformat"},
		"phgt":    dmapEntry{"phgt", dmapPrintUint, "dpap.imagepixelheight"},
		"picd":    dmapEntry{"picd", dmapPrintDate, "dpap.creationdate"},
		"pifs":    dmapEntry{"pifs", dmapPrintUint, "dpap.imagefilesize"},
		"pimf":    dmapEntry{"pimf", dmapPrintStr, "dpap.imagefilename"},
		"plsz":    dmapEntry{"plsz", dmapPrintUint, "dpap.imagelargefilesize"},
		"ppro":    dmapEntry{"ppro", dmapPrintVers, "dpap.protocolversion"},
		"prat":    dmapEntry{"prat", dmapPrintUint, "dpap.imagerating"},
		"pret":    dmapEntry{"pret", dmapPrintDict, "dpap.retryids"},
		"pwth":    dmapEntry{"pwth", dmapPrintUint, "dpap.imagepixelwidth"},
	}
}
