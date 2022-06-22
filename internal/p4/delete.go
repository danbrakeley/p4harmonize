package p4

import (
	"fmt"
	"strings"
)

// Delete marks one or more files for delete.
// Note that this will call `p4 delete`, which will immediately delete the local copy of the file.
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
	fnCleanup, filename, err := WriteTempFile("p4harmonize_delete_*.txt", strings.Join(paths, "\n"))
	if err != nil {
		return err
	}
	defer fnCleanup()

	return p.sh.Cmdf(`%s -x "%s" delete %s`, p.cmd(), filename, strings.Join(args, " ")).RunErr()
}
