package p4

import (
	"bufio"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var reClientName = regexp.MustCompile("^[a-zA-Z0-9_-]+$")

// CreateClient creates a new client with the given parameters
func (p *P4) CreateClient(clientname string, root string, stream string) error {
	if !reClientName.MatchString(clientname) {
		return fmt.Errorf("invalid client name: %s", clientname)
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for '%s': %w", root, err)
	}
	bashCmd := fmt.Sprintf(
		`%s --field "Root=%s" --field "Stream=%s" --field "View=%s/... //%s/..." client -o %s | %s client -i`,
		p.cmd(), absoluteRoot, stream, stream, clientname, clientname, p.cmd(),
	)
	err = p.sh.Cmd(bashCmd).BashErr()
	if err != nil {
		return fmt.Errorf("error creating client %s: %w", clientname, err)
	}
	return nil
}

// DeleteClient deletes an existing client spec that has no changelists or open files
func (p *P4) DeleteClient(clientname string) error {
	err := p.sh.Cmdf("%s client -d %s", p.cmd(), clientname).RunErr()
	if err != nil {
		return fmt.Errorf("error deleting client '%s': %w", p.Client, err)
	}
	return nil
}

// GetClientSpec requests the current client spec, and returns the resulting spec as a map of key/value pairs.
func (p *P4) GetClientSpec() (map[string]string, error) {
	var sb strings.Builder
	sb.Grow(1024)
	err := p.sh.Cmdf(`%s -z tag client -o`, p.cmd()).Out(&sb).RunErr()
	if err != nil {
		return nil, fmt.Errorf("error getting client %s: %w", p.Client, err)
	}
	return ParseSpec(sb.String()), nil
}

// ParseSpec takes in a string with a single spec formatted using -ztag, and
// returns a map with the key/value pairs in that spec
// For example:
//   ... Client super_client
//   ... Update 2021/09/16 22:30:29
//   ... Description first line
//
//   ... Root C:\Users\Super\Perforce
// Becomes:
//   map[string]string{
//     "Client": "super_client",
//     "Description": "first line\n",
//     "Root": "C:\\Users\\Super\\Perforce",
//     "Update": "2021/09/16 22:30:29",
//   }
func ParseSpec(spec string) map[string]string {
	out := make(map[string]string)
	var key, val string

	s := bufio.NewScanner(strings.NewReader(spec))
	for s.Scan() {
		line := s.Text()
		if !strings.HasPrefix(line, "... ") {
			val += "\n" + line
			continue
		}
		if len(key) > 0 {
			out[key] = val
		}
		i := strings.Index(line[4:], " ")
		if i == -1 {
			key = line[4:]
			val = ""
		} else {
			key = line[4 : 4+i]
			val = line[5+i:]
		}
	}

	if len(key) > 0 {
		out[key] = val
	}

	return out
}
