package p4

import (
	"fmt"
	"strconv"
	"strings"
)

// CreateEmptyChangelist creates a new changelist
func (p *P4) CreateEmptyChangelist(description string) (int64, error) {
	if strings.Contains(description, `"`) {
		// TODO: this is because of how we're building the change spec below.
		// This could be fixed in the future to remove this limitation, if there's ever a need.
		return 0, fmt.Errorf("double quotes not currently supported in changelist description")
	}

	// generate a changelist spec
	var clspec strings.Builder
	clspec.Grow(256)
	cmd := fmt.Sprintf(`%s --field "Description=%s" --field "Files=" change -o`, p.cmd(), description)
	if err := p.sh.Cmd(cmd).Out(&clspec).RunErr(); err != nil {
		return 0, fmt.Errorf("error building changelist spec: %w", err)
	}

	// feed the spec back into p4 to create the changelist
	var clnum strings.Builder
	clnum.Grow(64)
	specReader := strings.NewReader(clspec.String())
	if err := p.sh.Cmdf(`%s change -i`, p.cmd()).In(specReader).Out(&clnum).RunErr(); err != nil {
		return 0, fmt.Errorf("error creating changelist: %w", err)
	}

	clRaw := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(clnum.String()), "Change "), " created.")
	cl, err := strconv.ParseInt(clRaw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("unable to parse changelist number from '%s': %v", clnum.String(), err)
	}
	return cl, nil
}
