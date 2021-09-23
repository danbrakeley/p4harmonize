package p4

import (
	"fmt"
	"strings"
)

// Add adds a new file to the depot. Unlike other p4 commands, file paths
// passed into Add must not escape the reserved characters #, @, %, and *.
func (p *P4) Add(path string, opts ...Option) error {
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
	return p.sh.Cmdf(`%s add %s -f "%s"`, p.cmd(), strings.Join(args, " "), path).RunErr()
}
