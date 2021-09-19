package p4

import (
	"fmt"
	"strings"
)

// Delete marks a file in the depot for delete (which deletes any local copy of the file as well).
func (p *P4) Delete(path string, opts ...Option) error {
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
	return p.sh.Cmdf(`%s delete %s "%s"`, p.cmd(), strings.Join(args, " "), path).RunErr()
}
