package p4

import (
	"fmt"
	"os"
	"strings"
)

// Edit checks out an existing file from the depot. If your path includes any reserved
// characters (@#%*), you need to first escape your path with EscapePath.
func (p *P4) Edit(paths []string, opts ...Option) error {
	var args []string
	for _, o := range opts {
		switch ot := o.(type) {
		case oChangelist:
			if ot.CL > 0 {
				args = append(args, fmt.Sprintf("-c %d", ot.CL))
			}
		case oType:
			if len(ot.Type) > 0 {
				args = append(args, fmt.Sprintf(`-t %s`, ot.Type))
			}
		default:
			return fmt.Errorf("unrecognized option %s", o.String())
		}
	}

	// write paths to disk to avoid command line character limit
	tmpFilePattern := "p4harmonize_edit_*.txt"
	file, err := os.CreateTemp("", tmpFilePattern)
	if err != nil {
		return fmt.Errorf("Error creating temp file for pattern %s: %w", tmpFilePattern, err)
	}
	defer os.Remove(file.Name())
	file.WriteString(strings.Join(paths, "\n"))

	return p.sh.Cmdf(`%s -x "%s" edit %s`, p.cmd(), file.Name(), strings.Join(args, " ")).RunErr()
}
