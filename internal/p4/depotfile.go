package p4

import (
	"fmt"
	"sort"
	"strings"
)

type DepotFile struct {
	Path   string // relative to depot, ie 'Engine/foo', not '//UE4/Release/Engine/foo'
	Action string
	CL     string
	Type   string
}

// runAndParseDepotFiles calls the given command, which is expected to return a list of records, each
// with at least a depotFile, and optionally also a type, change, and action.
// The results are then sorted by Path (case-insensitive) and returned.
func (p *P4) runAndParseDepotFiles(cmd string) ([]DepotFile, error) {
	if !strings.Contains(cmd, "-ztag") && !strings.Contains(cmd, "-z tag") {
		return nil, fmt.Errorf(`missing "-z tag" in cmd: %s`, cmd)
	}

	streamDepth, err := p.StreamDepth()
	if err != nil {
		return nil, err
	}

	out := make([]DepotFile, 0, 1024*1024)
	var cur DepotFile
	var prefix string
	err = p.cmdAndScan(
		cmd,
		func(rawLine string) error {
			line := strings.TrimSpace(rawLine)

			// p4 -ztag uses an empty line to indicate the end of a record
			if len(line) == 0 {
				if len(cur.Path) != 0 {
					out = append(out, cur)
				}
				cur = DepotFile{}
				return nil
			}

			// otherwise, parse the fields
			switch {
			case len(line) < 5 || !strings.HasPrefix(line, "... "):
				return fmt.Errorf(`expected "... <tag>", but got: %s`, line)
			case strings.HasPrefix(line[4:], "depotFile"):
				raw := strings.TrimSpace(line[14:])
				if len(prefix) == 0 {
					var err error
					prefix, err = getDepotPrefix(raw, streamDepth)
					if err != nil {
						return fmt.Errorf(`error parsing depot prefix: %w`, err)
					}
				}
				cur.Path = strings.TrimPrefix(raw, prefix)
			case strings.HasPrefix(line[4:], "action"):
				cur.Action = strings.TrimSpace(line[10:])
			case strings.HasPrefix(line[4:], "change"):
				cur.CL = strings.TrimSpace(line[10:])
			case strings.HasPrefix(line[4:], "type"):
				cur.Type = strings.TrimSpace(line[8:])
			}

			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf(`error listing files: %w`, err)
	}

	// sort in-place, alphabetical, ignoring case
	sort.Sort(DepotFileCaseInsensitive(out))

	return out, nil
}

// getDepotPrefix returns the stream prefix given a line that includes the prefix and the stream depth
// For example: ("//a/b/c/d:foo", 2) would return "//a/b/"
func getDepotPrefix(line string, depth int) (string, error) {
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

// DepotFileCaseInsensitive allows sorting slices of DepotFiles by path, but ignoring case.
type DepotFileCaseInsensitive []DepotFile

func (x DepotFileCaseInsensitive) Len() int { return len(x) }
func (x DepotFileCaseInsensitive) Less(i, j int) bool {
	return strings.ToLower(x[i].Path) < strings.ToLower(x[j].Path)
}
func (x DepotFileCaseInsensitive) Swap(i, j int) { x[i], x[j] = x[j], x[i] }
