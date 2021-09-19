package p4

import (
	"fmt"
	"strings"
)

// Edit checks out an existing file from the depot.
func (p *P4) Edit(path string, opts ...Option) error {
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
	return p.sh.Cmdf(`%s edit %s "%s"`, p.cmd(), strings.Join(args, " "), path).RunErr()
}
