package p4

import (
	"fmt"
	"strings"

	"github.com/danbrakeley/bs"
)

type P4 struct {
	Port   string
	User   string
	Client string
}

func New(port, user, client string) P4 {
	return P4{
		Port:   port,
		User:   user,
		Client: client,
	}
}

// ListFiles requests all non-deleted files in the form "relative/path:filetype".
// For example: "Engine/Build/Build.version:text"
func (p P4) ListFiles() ([]string, error) {
	var sb strings.Builder
	sb.Grow(4 * 1024 * 1024) // start big to minimize reallocation

	err := bs.Cmdf(`%s -F %%depotFile%%:%%type%% files -e //%s/...`, p.cmd(), p.Client).Out(&sb).RunErr()
	if err != nil {
		return nil, fmt.Errorf(`error listing files: %w`, err)
	}

	lines := strings.Split(sb.String(), "\n")
	sortCaseInsensitive(lines)

	// cull blank lines (which should all have been sorted to the top)
	for len(strings.TrimSpace(lines[0])) == 0 {
		lines = lines[1:]
	}

	// find common prefix
	depth, err := p.StreamDepth()
	if err != nil {
		return nil, err
	}
	prefix, err := getStreamPrefix(lines[0], depth)
	if err != nil {
		return nil, err
	}

	// trim that prefix off every line
	for i := range lines {
		lines[i] = strings.TrimSpace(strings.TrimPrefix(lines[i], prefix))
	}

	return lines, nil
}

// StreamDepth requests a client's Stream, then parses it to determine the stream's depth.
// A stream named //foo/bar has a depth of 2, and //foo/bar/baz has a depth of 3.
func (p P4) StreamDepth() (int, error) {
	var sb strings.Builder
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

	return count, nil
}

// helpers

func (p P4) cmd() string {
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

// getStreamPrefix returns the stream prefix given a line that includes the prefix and the stream depth
// For example: ("//a/b/c/d:foo", 2) would return "//a/b/"
func getStreamPrefix(line string, depth int) (string, error) {
	if !strings.HasPrefix(line, "//") {
		return "", fmt.Errorf(`line "%s" does not begin with "//"`, line)
	}
	i := 2
	for depth > 0 {
		i += strings.Index(line[i:], "/")
		i++
		depth--
	}

	return line[:i], nil
}
