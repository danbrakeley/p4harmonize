package p4

import (
	"fmt"
	"strings"
)

func (p *P4) GetDepotSpec(name string) (map[string]string, error) {
	var sb strings.Builder
	sb.Grow(1024)
	err := p.sh.Cmdf(`%s -z tag depot -o %s`, p.cmd(), name).Out(&sb).RunErr()
	if err != nil {
		return nil, fmt.Errorf("error getting depot %s: %w", name, err)
	}
	return ParseSpec(sb.String()), nil
}

// CreateStreamDepot creates a depot with type "stream".
func (p *P4) CreateStreamDepot(name string) error {
	// generate a depot spec
	var b strings.Builder
	b.Grow(256)
	if err := p.sh.Cmdf(`%s --field "Type=stream" depot -o %s`, p.cmd(), name).Out(&b).RunErr(); err != nil {
		return fmt.Errorf("error building depot spec: %w", err)
	}

	// feed the spec back into p4 to create the depot
	specReader := strings.NewReader(b.String())
	if err := p.sh.Cmdf(`%s depot -i`, p.cmd()).In(specReader).RunErr(); err != nil {
		return fmt.Errorf("error creating depot: %w", err)
	}

	return nil
}
