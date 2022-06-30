package p4

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/danbrakeley/bsh"
)

type P4 struct {
	Port   string
	User   string
	Client string

	sh *bsh.Bsh

	streamMutex      sync.Mutex
	streamNameCache  string // read/write requires mutex lock
	streamDepthCache int    // read/write requires mutex lock
}

func New(sh *bsh.Bsh, port, user, client string) P4 {
	return P4{
		Port:   port,
		User:   user,
		Client: client,
		sh:     sh,
	}
}

// DisplayName just returns the Port at the moment. Should it do anything fancier?
func (p *P4) DisplayName() string {
	return p.Port
}

func (p *P4) SetStreamName(stream string) error {
	p.streamMutex.Lock()
	defer p.streamMutex.Unlock()
	depth, err := streamDepth(stream)
	if err != nil {
		return err
	}
	p.streamNameCache = stream
	p.streamDepthCache = depth
	return nil
}

func (p *P4) StreamInfo() (string, int, error) {
	p.streamMutex.Lock()
	defer p.streamMutex.Unlock()

	if len(p.streamNameCache) > 0 {
		return p.streamNameCache, p.streamDepthCache, nil
	}

	spec, err := p.GetClientSpec()
	if err != nil {
		return "", 0, fmt.Errorf("error getting stream name: %w", err)
	}

	stream, exists := spec["Stream"]
	if !exists || len(stream) == 0 {
		return "", 0, fmt.Errorf("client spec does not include Stream field")
	}

	depth, err := streamDepth(stream)
	if err != nil {
		return "", 0, err
	}

	p.streamNameCache = stream
	p.streamDepthCache = depth

	return stream, depth, nil
}

func streamDepth(stream string) (int, error) {
	count := -1
	for _, r := range stream {
		if r == '/' {
			count++
		}
	}
	if count < 1 {
		return 0, fmt.Errorf("unable to get stream depth of '%s'", stream)
	}
	return count, nil
}

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

// cmdAndScan streams the output of a cmd into a scanner, which calls the passed func for each line
func (p *P4) cmdAndScan(cmd string, fnEachLine func(line string) error) error {
	r, w := io.Pipe()
	chCmd := make(chan error)
	go func() {
		err := p.sh.Cmd(cmd).Out(w).RunErr()
		w.Close()
		chCmd <- err
	}()

	s := bufio.NewScanner(r)
	for s.Scan() {
		err := fnEachLine(s.Text())
		if err != nil {
			r.CloseWithError(err)
		}
	}
	err := <-chCmd
	if err == nil {
		err = s.Err()
	}
	return err
}

type DepotFile struct {
	Path   string // relative to depot, ie 'Engine/foo', not '//UE4/Release/Engine/foo'
	Action string
	CL     string
	Type   string
	Digest string
}

// DepotFileCaseInsensitive allows sorting slices of DepotFile by path, but ignoring case.
type DepotFileCaseInsensitive []DepotFile

func (x DepotFileCaseInsensitive) Len() int { return len(x) }
func (x DepotFileCaseInsensitive) Less(i, j int) bool {
	return strings.ToLower(x[i].Path) < strings.ToLower(x[j].Path)
}
func (x DepotFileCaseInsensitive) Swap(i, j int) { x[i], x[j] = x[j], x[i] }

// runAndParseDepotFiles calls the given command, which is expected to return a list of records, each
// with at least a depotFile, and optionally also a type, change, action, digest, headType, headChange,
// and headAction.
// The results are then sorted by Path (case-insensitive) and returned.
func (p *P4) runAndParseDepotFiles(cmd string) ([]DepotFile, error) {
	if !strings.Contains(cmd, "-ztag") && !strings.Contains(cmd, "-z tag") && !strings.Contains(cmd, "fstat") {
		return nil, fmt.Errorf("missing '-z tag' in non-fstat cmd: %s", cmd)
	}

	_, streamDepth, err := p.StreamInfo()
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
				return fmt.Errorf("expected '... <tag>', but got: %s", line)
			case strings.HasPrefix(line[4:], "depotFile"):
				raw := strings.TrimSpace(line[14:])
				if len(prefix) == 0 {
					var err error
					prefix, err = getDepotPrefix(raw, streamDepth)
					if err != nil {
						return fmt.Errorf("error parsing depot prefix: %w", err)
					}
				}
				// remove the prefix by length since the depot prefix may differ in case
				cur.Path = raw[len(prefix):]
			case strings.HasPrefix(line[4:], "action"):
				cur.Action = strings.TrimSpace(line[10:])
			case strings.HasPrefix(line[4:], "headAction"):
				cur.Action = strings.TrimSpace(line[14:])
			case strings.HasPrefix(line[4:], "change"):
				cur.CL = strings.TrimSpace(line[10:])
			case strings.HasPrefix(line[4:], "headChange"):
				cur.CL = strings.TrimSpace(line[14:])
			case strings.HasPrefix(line[4:], "type"):
				cur.Type = strings.TrimSpace(line[8:])
			case strings.HasPrefix(line[4:], "headType"):
				cur.Type = strings.TrimSpace(line[12:])
			case strings.HasPrefix(line[4:], "digest"):
				cur.Digest = strings.TrimSpace(line[10:])
			}

			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error listing files: %w", err)
	}

	// sort in-place, alphabetical, ignoring case
	sort.Sort(DepotFileCaseInsensitive(out))

	return out, nil
}

// getDepotPrefix returns the stream prefix given a line that includes the prefix and the stream depth
// For example: ("//a/b/c/d:foo", 2) would return "//a/b/"
func getDepotPrefix(line string, depth int) (string, error) {
	if !strings.HasPrefix(line, "//") {
		return "", fmt.Errorf("line '%s' does not begin with '//'", line)
	}
	i := 2
	for depth > 0 {
		i += strings.Index(line[i:], "/")
		i++
		depth--
	}

	return line[:i], nil
}

// Escaping reserved characters in file paths

func EscapePath(path string) string {
	size := len(path)
	for _, r := range path {
		switch r {
		case '@', '#', '*', '%':
			size += 2
		}
	}

	var sb strings.Builder
	sb.Grow(size)

	for _, r := range path {
		switch r {
		case '@':
			sb.WriteString("%40")
		case '#':
			sb.WriteString("%23")
		case '*':
			sb.WriteString("%2A")
		case '%':
			sb.WriteString("%25")
		default:
			sb.WriteRune(r)
		}
	}

	return sb.String()
}

func UnescapePath(path string) (string, error) {
	var sb strings.Builder
	sb.Grow(len(path))
	var escaped bool
	var start int

	for i, r := range path {
		if escaped {
			if i-start >= 2 {
				c, err := strconv.ParseInt(path[start+1:i+1], 16, 64)
				if err != nil {
					return "", fmt.Errorf("error parsing perforce-style escape code '%s': %w", path[start:i+1], err)
				}
				sb.WriteRune(rune(c))
				escaped = false
			}

			continue
		}

		if r == '%' {
			escaped = true
			start = i
			continue
		}

		sb.WriteRune(r)
	}

	if escaped {
		return "", fmt.Errorf("string ended before escaped character value in '%s'", path)
	}

	return sb.String(), nil
}

// WriteTempFile creates a temporary file then writes the passed contents to that file.
// To understand "filepattern", see the os.CreateTemp() documentation for the "pattern" argument.
// If there is no error in creating the file, then the returned func must be called
// when it is safe to delete the created temporary file.
func WriteTempFile(filepattern, contents string) (fnCleanup func(), filename string, err error) {
	file, err := os.CreateTemp("", filepattern)
	if err != nil {
		return nil, "", fmt.Errorf("Error creating temp file for pattern %s: %w", filepattern, err)
	}
	defer file.Close()

	_, err = file.WriteString(contents)
	if err != nil {
		return nil, "", fmt.Errorf("Error writing temp file for pattern %s: %w", filepattern, err)
	}

	name := file.Name()
	return func() { os.Remove(name) }, name, nil
}
