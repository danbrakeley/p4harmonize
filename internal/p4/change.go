package p4

import (
	"fmt"
	"strconv"
	"strings"
)

// CreateChangelist creates a new changelist
func (p *P4) CreateChangelist(description string) (int64, error) {
	var sb strings.Builder
	sb.Grow(128)
	bashCmd := fmt.Sprintf(
		`%s --field "Description=%s" --field "Files=" change -o | %s change -i`,
		p.cmd(), description, p.cmd(),
	)
	err := p.sh.Cmd(bashCmd).Out(&sb).BashErr()
	if err != nil {
		return 0, fmt.Errorf("error creating changelist: %w", err)
	}
	clRaw := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(sb.String()), "Change "), " created.")
	cl, err := strconv.ParseInt(clRaw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("changelist was created, but unable to parse '%s': %v", clRaw, err)
	}
	return cl, nil
}
