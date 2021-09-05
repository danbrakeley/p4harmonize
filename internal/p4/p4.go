package p4

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/danbrakeley/bs"
)

type P4 struct {
	Port   string
	User   string
	Client string

	// derived values
	displayName string

	streamDepthMutex sync.Mutex
	streamDepth      int
}

func New(port, user, client string) P4 {
	// try to find the hostname without any protocol prefix or port suffix
	name := port
	s := strings.Split(port, ":")
	i := len(s) - 1
	if i > 0 {
		_, err := strconv.ParseInt(s[i], 10, 32)
		if err == nil {
			i--
		}
		name = s[i]
	}

	return P4{
		Port:        port,
		User:        user,
		Client:      client,
		displayName: name,
	}
}

func (p *P4) DisplayName() string {
	return p.displayName
}

// OpenedFiles calls p4 opened and returns the results.
// Order of resulting slice is alphabetical by Path, ignoring case.
func (p *P4) OpenedFiles() ([]DepotFile, error) {
	return p.runAndParseDepotFiles(fmt.Sprintf(`%s -z tag opened -a -C %s`, p.cmd(), p.Client))
}

// DepotFiles does a "files -e" and returns the results.
// Order of resulting slice is alphabetical by Path, ignoring case.
func (p *P4) DepotFiles() ([]DepotFile, error) {
	return p.runAndParseDepotFiles(fmt.Sprintf(`%s -z tag files -e //%s/...`, p.cmd(), p.Client))
}

// StreamDepth requests a client's Stream, then parses it to determine the stream's depth.
// A stream named //foo/bar has a depth of 2, and //foo/bar/baz has a depth of 3.
func (p *P4) StreamDepth() (int, error) {
	p.streamDepthMutex.Lock()
	defer p.streamDepthMutex.Unlock()

	if p.streamDepth > 0 {
		return p.streamDepth, nil
	}

	var sb strings.Builder
	sb.Grow(2 * 1024)
	err := bs.Cmdf(`%s -z tag client -o`, p.cmd()).Out(&sb).RunErr()
	if err != nil {
		return 0, fmt.Errorf(`error viewing workspace "%s": %w`, p.Client, err)
	}

	stream := getFieldFromSpec(sb.String(), "Stream")
	if len(stream) == 0 {
		return 0, fmt.Errorf(`stream name not found for client "%s"`, p.Client)
	}

	count := -1
	for _, r := range stream {
		if r == '/' {
			count++
		}
	}

	if count < 1 {
		return 0, fmt.Errorf(`unable to parse depth from stream "%s"`, stream)
	}

	p.streamDepth = count

	return count, nil
}

// helpers

func (p *P4) cmd() string {
	out := strings.Builder{}
	out.WriteString("p4")
	if len(p.Port) > 0 {
		out.WriteString(" -p ")
		out.WriteString(p.Port)
	}
	if len(p.User) > 0 {
		out.WriteString(" -u ")
		out.WriteString(p.User)
	}
	if len(p.Client) > 0 {
		out.WriteString(" -c ")
		out.WriteString(p.Client)
	}
	return out.String()
}

// getFieldFromSpec extracts the value of a field from a perforce spec that was formatted via -z tag
func getFieldFromSpec(spec, field string) string {
	lines := strings.Split(spec, "\n")
	for _, line := range lines {
		prefix := fmt.Sprintf("... %s ", field)
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}
