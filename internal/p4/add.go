package p4

import (
	"fmt"
	"os"
	"strings"
)

// Add adds new files to the depot. Unlike other p4 commands, file paths
// passed into Add must not escape the reserved characters #, @, %, and *.
func (p *P4) Add(paths []string, opts ...Option) error {
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
	file, err := os.CreateTemp("", "p4harmonize_add_*.txt")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	file.WriteString(strings.Join(paths, "\n"))

	return p.sh.Cmdf(`%s -x "%s" add %s -If`, p.cmd(), file.Name(), strings.Join(args, " ")).RunErr()
}
