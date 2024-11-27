package p4

import (
	"fmt"
	"strings"
)

// https://help.perforce.com/helix-core/server-apps/cmdref/2024.2/Content/CmdRef/p4_depot.html
type DepotType string

const (
	Local DepotType = "local"
	// Remote DepotType = "remote"
	Stream DepotType = "stream"
	// Spec DepotType = "spec"
	// Unload DepotType = "unload"
	// Archive DepotType = "archive"
	// Tangent DepotType = "tangent"
	// Graph DepotType = "graph"
	// Trait DepotType = "trait"
)

// CreateDepot creates a depot.
func (p *P4) CreateDepot(name string, typ DepotType) error {
	// generate a depot spec
	var b strings.Builder
	b.Grow(256)
	cmd := fmt.Sprintf(`%s --field "Type=%s" depot -o %s`, p.cmd(), string(typ), name)
	if err := p.sh.Cmd(cmd).Out(&b).RunErr(); err != nil {
		return fmt.Errorf("error building depot spec: %w", err)
	}

	// feed the spec back into p4 to create the depot
	specReader := strings.NewReader(b.String())
	if err := p.sh.Cmdf(`%s depot -i`, p.cmd()).In(specReader).RunErr(); err != nil {
		return fmt.Errorf("error creating depot: %w", err)
	}

	return nil
}
