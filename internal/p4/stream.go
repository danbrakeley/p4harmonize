package p4

import (
	"fmt"
	"strings"
)

// CreateMainlineStream creates a new stream with type mainline and whose full stream path is //depot/name
func (p *P4) CreateMainlineStream(depot, name string) error {
	// generate a stream spec
	var b strings.Builder
	b.Grow(256)
	cmd := fmt.Sprintf(`%s --field "Type=mainline" stream -o //%s/%s`, p.cmd(), depot, name)
	if err := p.sh.Cmd(cmd).Out(&b).RunErr(); err != nil {
		return fmt.Errorf("error building stream spec: %w", err)
	}

	// feed the spec back into p4 to create the stream
	specReader := strings.NewReader(b.String())
	if err := p.sh.Cmdf(`%s stream -i`, p.cmd()).In(specReader).RunErr(); err != nil {
		return fmt.Errorf("error creating mainline stream: %w", err)
	}

	return nil
}
