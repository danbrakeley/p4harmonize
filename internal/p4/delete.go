package p4

import (
	"fmt"
	"os"
	"strings"
)

// Delete marks a file in the depot for delete (which deletes any local copy of the file as well).
func (p *P4) Delete(paths []string, opts ...Option) error {
	var args []string
	for _, o := range opts {
		switch ot := o.(type) {
		case oChangelist:
			if ot.CL > 0 {
				args = append(args, fmt.Sprintf("-c %d", ot.CL))
			}
		default:
			return fmt.Errorf("unrecognized option %s", o.String())
		}
	}

	// write paths to disk to avoid command line character limit
	tmpFilePattern := "p4harmonize_delete_*.txt"
	file, err := os.CreateTemp("", tmpFilePattern)
	if err != nil {
		return fmt.Errorf("Error creating temp file for pattern %s: %w", tmpFilePattern, err)
	}
	defer os.Remove(file.Name())
	file.WriteString(strings.Join(paths, "\n"))

	return p.sh.Cmdf(`%s -x "%s" delete %s`, p.cmd(), file.Name(), strings.Join(args, " ")).RunErr()
}
