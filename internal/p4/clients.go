package p4

import (
	"fmt"
	"strings"
)

// ListClients returns a list of client names for the current user
func (p *P4) ListClients() ([]string, error) {
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
