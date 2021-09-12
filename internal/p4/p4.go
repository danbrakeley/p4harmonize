package p4

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/danbrakeley/bsh"
)

type P4 struct {
	Port   string
	User   string
	Client string

	sh *bsh.Bsh

	streamMutex sync.Mutex
	streamName  string // read/write requires mutex lock
	streamDepth int    // read/write requires mutex lock
}

func New(sh *bsh.Bsh, port, user, client string) P4 {
	return P4{
		Port:   port,
		User:   user,
		Client: client,
		sh:     sh,
	}
}

func (p *P4) DisplayName() string {
	return p.Port
}

func (p *P4) SetStreamName(stream string) error {
	p.streamMutex.Lock()
	defer p.streamMutex.Unlock()
	depth, err := streamDepthFromName(stream)
	if err != nil {
		return err
	}
	p.streamName = stream
	p.streamDepth = depth
	return nil
}

// SyncLatest runs p4 sync ...#head
func (p *P4) SyncLatest() error {
	err := p.sh.Cmdf(`%s sync //%s/...#head`, p.cmd(), p.Client).RunErr()
	if err != nil {
		return fmt.Errorf(`error syncing %s to head: %w`, p.Client, err)
	}
	return nil
}

// SyncLatestNoDownload runs "p4 sync -k ...#head" which will:
// "Keep existing workspace files; update the have list without updating the client workspace"
func (p *P4) SyncLatestNoDownload() error {
	err := p.sh.Cmdf(`%s sync -k //%s/...#head`, p.cmd(), p.Client).Out(nil).RunErr()
	if err != nil {
		return fmt.Errorf(`error fake-syncing %s to head: %w`, p.Client, err)
	}
	return nil
}

// Clients returns a list of client names
func (p *P4) Clients() ([]string, error) {
	var out []string
	err := p.cmdAndScan(
		fmt.Sprintf(`%s -F %%domainName%% clients -u %s`, p.cmd(), p.User),
		func(line string) error {
			out = append(out, strings.TrimSpace(line))
			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf(`error listing clients: %w`, err)
	}
	return out, nil
}

var reClientName = regexp.MustCompile("^[a-zA-Z0-9_-]+$")

// CreateClient creates a new client with the given parameters
func (p *P4) CreateClient(clientname string, root string, stream string) error {
	if !reClientName.MatchString(clientname) {
		return fmt.Errorf(`invalid client name "%s"`, clientname)
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf(`failed to get absolute path for "%s": %w`, root, err)
	}
	bashCmd := fmt.Sprintf(
		`%s --field "Root=%s" --field "Stream=%s" --field "View=%s/... //%s/..." client -o %s | %s client -i`,
		p.cmd(), absoluteRoot, stream, stream, clientname, clientname, p.cmd(),
	)
	err = p.sh.Cmd(bashCmd).BashErr()
	if err != nil {
		return fmt.Errorf(`error creating client "%s": %w`, p.Client, err)
	}
	return nil
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
	p.streamMutex.Lock()
	defer p.streamMutex.Unlock()

	if p.streamDepth > 0 {
		return p.streamDepth, nil
	}

	var stream string
	if len(p.streamName) == 0 {
		var sb strings.Builder
		sb.Grow(2 * 1024)
		err := p.sh.Cmdf(`%s -z tag client -o`, p.cmd()).Out(&sb).RunErr()
		if err != nil {
			return 0, fmt.Errorf(`error viewing workspace "%s": %w`, p.Client, err)
		}

		stream = getFieldFromSpec(sb.String(), "Stream")
		if len(stream) == 0 {
			return 0, fmt.Errorf(`stream name not found for client "%s"`, p.Client)
		}
	}

	depth, err := streamDepthFromName(stream)
	if err != nil {
		p.streamDepth = depth
		p.streamName = stream
	}

	return depth, err
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

func streamDepthFromName(stream string) (int, error) {
	count := -1
	for _, r := range stream {
		if r == '/' {
			count++
		}
	}
	if count < 1 {
		return 0, fmt.Errorf(`unable to get stream depth of "%s"`, stream)
	}
	return count, nil
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
