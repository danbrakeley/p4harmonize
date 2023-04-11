package p4

import (
	"fmt"
	"strings"
)

type CaseType uint8

const (
	CaseUnknown     CaseType = iota
	CaseInsensitive CaseType = iota
	CaseSensitive   CaseType = iota
)

type Info struct {
	CaseHandling CaseType
}

// Info runs the info command against the server.
func (p *P4) Info() (Info, error) {
	var info Info

	err := p.cmdAndScan(
		fmt.Sprintf("%s info", p.cmd()),
		func(rawLine string) error {
			line := strings.TrimSpace(rawLine)
			if strings.HasPrefix(line, "Case Handling:") {
				value := strings.TrimSpace(strings.TrimPrefix(line, "Case Handling:"))
				switch value {
				case "insensitive":
					info.CaseHandling = CaseInsensitive
				case "sensitive":
					info.CaseHandling = CaseSensitive
				}
			}
			return nil
		},
	)

	if err != nil {
		return Info{}, err
	}

	return info, nil
}
