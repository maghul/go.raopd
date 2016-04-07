package raopd

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testDmap(t *testing.T) *dmap {
	data := []byte{
		0x6d, 0x6c, 0x69, 0x74, 0x00, 0x00, 0x00, 0xa6, 0x6d, 0x70, 0x65, 0x72, 0x00, 0x00, 0x00, 0x08,
		0x60, 0x14, 0xa4, 0x10, 0x49, 0x19, 0x6f, 0xe0, 0x61, 0x73, 0x61, 0x6c, 0x00, 0x00, 0x00, 0x14,
		0x4d, 0x61, 0x67, 0x69, 0x63, 0x61, 0x6c, 0x20, 0x4d, 0x79, 0x73, 0x74, 0x65, 0x72, 0x79, 0x20,
		0x54, 0x6f, 0x75, 0x72, 0x61, 0x73, 0x61, 0x72, 0x00, 0x00, 0x00, 0x0b, 0x54, 0x68, 0x65, 0x20,
		0x42, 0x65, 0x61, 0x74, 0x6c, 0x65, 0x73, 0x61, 0x73, 0x63, 0x70, 0x00, 0x00, 0x00, 0x00, 0x61,
		0x73, 0x67, 0x6e, 0x00, 0x00, 0x00, 0x04, 0x52, 0x6f, 0x63, 0x6b, 0x6d, 0x69, 0x6e, 0x6d, 0x00,
		0x00, 0x00, 0x0f, 0x49, 0x20, 0x41, 0x6d, 0x20, 0x54, 0x68, 0x65, 0x20, 0x57, 0x61, 0x6c, 0x72,
		0x75, 0x73, 0x61, 0x73, 0x74, 0x6e, 0x00, 0x00, 0x00, 0x02, 0x00, 0x06, 0x61, 0x73, 0x74, 0x63,
		0x00, 0x00, 0x00, 0x02, 0x00, 0x0b, 0x61, 0x73, 0x64, 0x6e, 0x00, 0x00, 0x00, 0x02, 0x00, 0x01,
		0x61, 0x73, 0x64, 0x6b, 0x00, 0x00, 0x00, 0x01, 0x00, 0x63, 0x61, 0x70, 0x73, 0x00, 0x00, 0x00,
		0x01, 0x01, 0x61, 0x73, 0x74, 0x6d, 0x00, 0x00, 0x00, 0x04, 0x00, 0x04, 0x3a, 0x8a}

	dm, err := newDmap(bytes.NewBuffer(data))
	assert.NoError(t, err)
	return dm
}

func testDmap2(t *testing.T) *dmap {
	data := []byte{109, 108, 105, 116, 0, 0, 4, 102, 109, 105, 107, 100, 0, 0, 0, 1, 2, 97, 115, 97, 108, 0, 0, 0, 10, 78, 105, 110, 106, 97, 32, 84, 117, 110, 97, 97, 115, 97, 114, 0, 0, 0, 10, 77, 114, 46, 32, 83, 99, 114, 117, 102, 102, 97, 115, 98, 114, 0, 0, 0, 2, 0, 192, 97, 115, 99, 109, 0, 0, 0, 18, 78, 105, 110, 106, 97, 32, 84, 117, 110, 101, 32, 82, 101, 99, 111, 114, 100, 115, 97, 115, 99, 111, 0, 0, 0, 1, 0, 97, 115, 99, 112, 0, 0, 0, 25, 65, 46, 32, 67, 97, 114, 116, 104, 121, 32, 97, 110, 100, 32, 65, 46, 32, 75, 105, 110, 103, 115, 108, 111, 119, 109, 101, 105, 97, 0, 0, 0, 4, 87, 3, 251, 202, 97, 115, 100, 97, 0, 0, 0, 4, 87, 3, 251, 202, 109, 101, 105, 112, 0, 0, 0, 4, 131, 218, 51, 96, 97, 115, 112, 108, 0, 0, 0, 4, 131, 218, 51, 96, 97, 115, 100, 109, 0, 0, 0, 4, 74, 92, 24, 111, 97, 115, 100, 99, 0, 0, 0, 2, 0, 0, 97, 115, 100, 110, 0, 0, 0, 2, 0, 0, 97, 115, 101, 113, 0, 0, 0, 0, 97, 115, 103, 110, 0, 0, 0, 10, 69, 108, 101, 99, 116, 114, 111, 110, 105, 99, 97, 115, 100, 116, 0, 0, 0, 15, 77, 80, 69, 71, 32, 97, 117, 100, 105, 111, 32, 102, 105, 108, 101, 97, 115, 114, 118, 0, 0, 0, 1, 0, 97, 115, 115, 114, 0, 0, 0, 4, 0, 0, 172, 68, 97, 115, 115, 122, 0, 0, 0, 4, 0, 128, 100, 241, 97, 115, 115, 116, 0, 0, 0, 4, 0, 0, 0, 0, 97, 115, 115, 112, 0, 0, 0, 4, 0, 0, 0, 0, 97, 115, 116, 109, 0, 0, 0, 4, 0, 5, 79, 151, 97, 115, 116, 99, 0, 0, 0, 2, 0, 0, 97, 115, 116, 110, 0, 0, 0, 2, 0, 1, 97, 115, 117, 114, 0, 0, 0, 1, 0, 97, 115, 121, 114, 0, 0, 0, 2, 7, 216, 97, 115, 102, 109, 0, 0, 0, 3, 109, 112, 51, 109, 105, 105, 100, 0, 0, 0, 4, 0, 0, 0, 78, 109, 105, 110, 109, 0, 0, 0, 7, 75, 97, 108, 105, 109, 98, 97, 109, 112, 101, 114, 0, 0, 0, 8, 28, 205, 9, 116, 229, 9, 60, 13, 97, 115, 100, 98, 0, 0, 0, 1, 0, 97, 101, 78, 86, 0, 0, 0, 4, 0, 0, 0, 0, 97, 115, 100, 107, 0, 0, 0, 1, 0, 97, 115, 98, 116, 0, 0, 0, 2, 0, 0, 97, 103, 114, 112, 0, 0, 0, 0, 97, 115, 99, 100, 0, 0, 0, 4, 109, 112, 101, 103, 97, 115, 99, 115, 0, 0, 0, 4, 0, 0, 0, 3, 97, 101, 80, 67, 0, 0, 0, 1, 0, 97, 115, 99, 116, 0, 0, 0, 0, 97, 115, 99, 110, 0, 0, 0, 0, 97, 115, 99, 114, 0, 0, 0, 1, 0, 97, 101, 72, 86, 0, 0, 0, 1, 0, 97, 101, 77, 75, 0, 0, 0, 1, 1, 97, 101, 83, 78, 0, 0, 0, 0, 97, 101, 69, 78, 0, 0, 0, 0, 97, 101, 69, 83, 0, 0, 0, 4, 0, 0, 0, 0, 97, 101, 83, 85, 0, 0, 0, 4, 0, 0, 0, 0, 97, 101, 71, 72, 0, 0, 0, 4, 2, 0, 0, 3, 97, 101, 71, 68, 0, 0, 0, 4, 0, 0, 2, 232, 97, 101, 71, 85, 0, 0, 0, 8, 0, 0, 0, 0, 0, 234, 49, 8, 97, 101, 71, 82, 0, 0, 0, 8, 0, 0, 0, 0, 0, 127, 98, 157, 97, 101, 71, 69, 0, 0, 0, 4, 0, 0, 2, 16, 97, 115, 97, 97, 0, 0, 0, 10, 77, 114, 46, 32, 83, 99, 114, 117, 102, 102, 97, 115, 103, 112, 0, 0, 0, 1, 0, 109, 101, 120, 116, 0, 0, 0, 2, 0, 1, 97, 115, 101, 100, 0, 0, 0, 2, 0, 1, 97, 115, 104, 112, 0, 0, 0, 1, 1, 97, 115, 115, 110, 0, 0, 0, 0, 97, 115, 115, 97, 0, 0, 0, 0, 97, 115, 115, 108, 0, 0, 0, 0, 97, 115, 115, 117, 0, 0, 0, 0, 97, 115, 115, 99, 0, 0, 0, 0, 97, 115, 115, 115, 0, 0, 0, 0, 97, 115, 98, 107, 0, 0, 0, 1, 0, 97, 115, 112, 117, 0, 0, 0, 0, 97, 101, 67, 82, 0, 0, 0, 0, 97, 115, 97, 105, 0, 0, 0, 8, 197, 140, 14, 24, 126, 33, 0, 156, 97, 115, 108, 115, 0, 0, 0, 8, 0, 0, 0, 0, 0, 128, 100, 241, 97, 101, 83, 69, 0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 0, 0, 97, 101, 68, 86, 0, 0, 0, 4, 0, 0, 0, 0, 97, 101, 68, 80, 0, 0, 0, 4, 0, 0, 0, 0, 97, 101, 68, 82, 0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 0, 0, 97, 101, 78, 68, 0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 0, 0, 97, 101, 75, 49, 0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 0, 0, 97, 101, 75, 50, 0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 0, 0, 97, 101, 68, 76, 0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 0, 0, 97, 101, 70, 65, 0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 0, 0, 97, 101, 88, 68, 0, 0, 0, 0, 97, 101, 77, 107, 0, 0, 0, 4, 0, 0, 0, 1, 97, 101, 77, 88, 0, 0, 0, 0, 97, 115, 112, 99, 0, 0, 0, 4, 0, 0, 0, 0, 97, 115, 114, 105, 0, 0, 0, 8, 33, 181, 239, 71, 136, 101, 105, 0, 97, 101, 67, 83, 0, 0, 0, 4, 0, 0, 233, 220, 97, 115, 107, 112, 0, 0, 0, 4, 0, 0, 0, 1, 97, 115, 97, 99, 0, 0, 0, 2, 0, 1, 97, 115, 107, 100, 0, 0, 0, 4, 87, 3, 252, 35, 109, 100, 115, 116, 0, 0, 0, 1, 1, 97, 115, 101, 115, 0, 0, 0, 1, 0, 97, 115, 114, 115, 0, 0, 0, 1, 0, 97, 115, 108, 114, 0, 0, 0, 1, 0, 97, 115, 97, 115, 0, 0, 0, 1, 32, 97, 101, 71, 115, 0, 0, 0, 1, 0, 97, 101, 108, 115, 0, 0, 0, 1, 0, 97, 106, 97, 108, 0, 0, 0, 1, 0, 97, 106, 99, 65, 0, 0, 0, 1, 0}

	dm, err := newDmap(bytes.NewBuffer(data))
	assert.NoError(t, err)
	return dm
}

func crlf(in string) string {
	return strings.Replace(in, "\n", "\r\n", -1)
}

func TestDmapToJson(t *testing.T) {
	json := testDmap(t).String("JSON")

	expected := `{
    "dmap.listingitem": {
        "dmap.persistentid": 6923338917028130784,
        "daap.songalbum": "Magical Mystery Tour",
        "daap.songartist": "The Beatles",
        "daap.songcomposer": "",
        "daap.songgenre": "Rock",
        "dmap.itemname": "I Am The Walrus",
        "daap.songtracknumber": 6,
        "daap.songtrackcount": 11,
        "daap.songdiscnumber": 1,
        "daap.songdatakind": 0,
        "dacp.playerstate": 1,
        "daap.songtime": 277130
    }
}
`
	assert.Equal(t, crlf(expected), json)
}

func TestDmapToXML(t *testing.T) {
	json := testDmap(t).String("XML")

	expected := `<?xml version="1.0" encoding="UTF-8"?>
<dmap.listingitem>
    <dmap.persistentid>6923338917028130784</dmap.persistentid>
    <daap.songalbum>Magical Mystery Tour</daap.songalbum>
    <daap.songartist>The Beatles</daap.songartist>
    <daap.songcomposer></daap.songcomposer>
    <daap.songgenre>Rock</daap.songgenre>
    <dmap.itemname>I Am The Walrus</dmap.itemname>
    <daap.songtracknumber>6</daap.songtracknumber>
    <daap.songtrackcount>11</daap.songtrackcount>
    <daap.songdiscnumber>1</daap.songdiscnumber>
    <daap.songdatakind>0</daap.songdatakind>
    <dacp.playerstate>1</dacp.playerstate>
    <daap.songtime>277130</daap.songtime>
</dmap.listingitem>
`
	assert.Equal(t, crlf(expected), json)
}

func TestDmap2ToJson(t *testing.T) {
	json := testDmap2(t).String("JSON")

	expected := `{
    "dmap.listingitem": {
        "dmap.itemkind": 2,
        "daap.songalbum": "Ninja Tuna",
        "daap.songartist": "Mr. Scruff",
        "daap.songbitrate": 192,
        "daap.songcomment": "Ninja Tune Records",
        "daap.songcompilation": 0,
        "daap.songcomposer": "A. Carthy and A. Kingslow",
        "dmap.itemdateadded": 1459878858,
        "daap.songdateadded": "2016-04-05 17:54:18 +0000 UTC",
        "dmap.itemdateplayed": 2212115296,
        "daap.songdateplayed": "2040-02-06 04:28:16 +0000 UTC",
        "daap.songdatemodified": "2009-07-14 05:32:31 +0000 UTC",
        "daap.songdisccount": 0,
        "daap.songdiscnumber": 0,
        "daap.songeqpreset": "",
        "daap.songgenre": "Electronic",
        "daap.songdescription": "MPEG audio file",
        "daap.songrelativevolume": 0,
        "daap.songsamplerate": 44100,
        "daap.songsize": 8414449,
        "daap.songstarttime": 0,
        "daap.songstoptime": 0,
        "daap.songtime": 348055,
        "daap.songtrackcount": 0,
        "daap.songtracknumber": 1,
        "daap.songuserrating": 0,
        "daap.songyear": 2008,
        "daap.songformat": "mp3",
        "dmap.itemid": 78,
        "dmap.itemname": "Kalimba",
        "dmap.persistentid": 2075325400951110669,
        "daap.songdisabled": 0,
        "com.apple.itunes.norm-volume": 0,
        "daap.songdatakind": 0,
        "daap.songbeatsperminute": 0,
        "daap.songgrouping": "",
        "daap.songcodectype": 1836082535,
        "daap.songcodecsubtype": 3,
        "com.apple.itunes.is-podcast": 0,
        "daap.songcategory": "",
        "daap.songcontentdescription": "",
        "daap.songcontentrating": 0,
        "com.apple.itunes.has-video": 0,
        "com.apple.itunes.mediakind": 1,
        "com.apple.itunes.series-name": "",
        "com.apple.itunes.episode-num-str": "",
        "com.apple.itunes.episode-sort": 0,
        "com.apple.itunes.season-num": 0,
        "com.apple.itunes.gapless-heur": 33554435,
        "com.apple.itunes.gapless-enc-dr": 744,
        "com.apple.itunes.gapless-dur": 15347976,
        "com.apple.itunes.gapless-resy": 8348317,
        "com.apple.itunes.gapless-enc-del": 528,
        "daap.songalbumartist": "Mr. Scruff",
        "daap.songgapless": 0,
        "dmap.objectextradata": 1,
        "daap.songextradata": 1,
        "daap.songhasbeenplayed": 1,
        "daap.sortname": "",
        "daap.sortartist": "",
        "daap.sortalbumartist": "",
        "daap.sortalbum": "",
        "daap.sortcomposer": "",
        "daap.sortseriesname": "",
        "daap.bookmarkable": 0,
        "daap.songpodcasturl": "",
        "com.apple.itunes.content-rating": "",
        "daap.songalbumid": -4211976053140160356,
        "daap.songlongsize": 8414449,
        "com.apple.itunes.store-pers-id": 0,
        "com.apple.itunes.drm-versions": 0,
        "com.apple.itunes.drm-platform-id": 0,
        "com.apple.itunes.drm-user-id": 0,
        "com.apple.itunes.non-drm-user-id": 0,
        "com.apple.itunes.drm-key1-id": 0,
        "com.apple.itunes.drm-key2-id": 0,
        "com.apple.itunes.drm-downloader-user-id": 0,
        "com.apple.itunes.drm-family-id": 0,
        "com.apple.itunes.xid": "",
        "com.apple.itunes.extended-media-kind": 1,
        "com.apple.itunes.movie-info-xml": "",
        "daap.songuserplaycount": 0,
        "daap.songartistid": 2429110664546314496,
        "com.apple.itunes.artworkchecksum": 59868,
        "daap.songuserskipcount": 1,
        "daap.songartworkcount": 1,
        "daap.songlastskipdate": "2016-04-05 17:55:47 +0000 UTC",
        "dmap.downloadstatus": 1,
        "daap.songexcludefromshuffle": 0,
        "daap.songuserratingstatus": 0,
        "daap.songalbumuserrating": 0,
        "daap.songalbumuserratingstatus": 32,
        "com.apple.itunes.can-be-genius-seed": 0,
        "com.apple.itunes.liked-state": 0,
        "com.apple.itunes.store.album-liked-state": 0,
        "com.apple.itunes.store.show-composer-as-artist": 0
    }
}
`
	assert.Equal(t, crlf(expected), json)
}
