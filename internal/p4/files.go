package p4

import (
	"fmt"
)

// ListDepotFiles runs "p4 fstat" and parses the results into a slice of DepotFile structs.
// Order of resulting slice is alphabetical by Path, ignoring case.
func (p *P4) ListDepotFiles() ([]DepotFile, error) {
	return p.runAndParseDepotFiles(fmt.Sprintf(`%s fstat -T depotFile,headAction,headChange,headType,digest -Ol -F '^(headAction=move/delete | headAction=purge | headAction=archive | headAction=delete)' //%s/...`, p.cmd(), p.Client))
}
