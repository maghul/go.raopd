package raopd

import (
	"bufio"
	"io"
	"strings"
)

type sdp map[string]string

func makeSdpRecords(r io.Reader) sdp {
	s := make(map[string]string)

	br := bufio.NewReader(r)
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.Trim(line, " \r\n")
		if line == "" {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if kv[0] == "a" {
			kv = strings.SplitN(line, ":", 2)
			s[kv[0]] = kv[1]
		} else {
			s[kv[0]] = kv[1]
		}
	}
	return s
}
