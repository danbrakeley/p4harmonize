package p4

import (
	"fmt"
	"strings"
)

// Move changes the path (including capitalization changes on case sensative servers) and filetype of a file in the depot.
func (p *P4) Move(from string, to string, opts ...Option) error {
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
		case oKeep:
			args = append(args, "-k")
		default:
			return fmt.Errorf("unrecognized option %s", o.String())
		}
	}
	return p.sh.Cmdf(`%s move %s "%s" "%s"`, p.cmd(), strings.Join(args, " "), from, to).RunErr()
}
