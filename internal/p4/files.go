package p4

import (
	"fmt"
)

// ListDepotFiles runs "p4 files -e" and parses the results into a slice of DepotFile structs.
// Order of resulting slice is alphabetical by Path, ignoring case.
func (p *P4) ListDepotFiles() ([]DepotFile, error) {
	return p.runAndParseDepotFiles(fmt.Sprintf(`%s -z tag files -e //%s/...`, p.cmd(), p.Client))
}
