package p4

import (
	"fmt"
	"strings"
)

// RevertUnchanged checks out an existing file from the depot.
func (p *P4) RevertUnchanged(root string, opts ...Option) error {
	var args []string
	for _, o := range opts {
		switch ot := o.(type) {
		case oChangelist:
			if ot.CL > 0 {
				args = append(args, fmt.Sprintf("-c %d", ot.CL))
			}
		case oKeep:
			args = append(args, "-k")
		default:
			return fmt.Errorf("unrecognized option %s", o.String())
		}
	}
	return p.sh.Cmdf(`%s revert -a %s "%s"`, p.cmd(), strings.Join(args, " "), root).RunErr()
}
