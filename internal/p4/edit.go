package p4

import (
	"fmt"
	"strings"
)

// Edit checks out one or more existing file(s) from the depot. If your path includes any
// reserved characters (@#%*), you need to first escape your path with EscapePath.
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
	fnCleanup, filename, err := WriteTempFile("p4harmonize_edit_*.txt", strings.Join(paths, "\n"))
	if err != nil {
		return err
	}
	defer fnCleanup()

	return p.sh.Cmdf(`%s -x "%s" edit %s`, p.cmd(), filename, strings.Join(args, " ")).RunErr()
}
