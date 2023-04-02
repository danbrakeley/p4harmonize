package p4

import (
	"fmt"
	"strings"
)

// Add adds new files to the depot. Unlike other p4 commands, file paths
// passed into Add must not escape the reserved characters #, @, %, and *.
// You can override this behavior by passing the AllowWildcards option.
func (p *P4) Add(paths []string, opts ...Option) error {
	args := make([]string, 0, len(opts)+1)
	allowWildcards := false
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
		case oDoNotIgnore:
			args = append(args, "-I")
		case oAllowWildcards:
			allowWildcards = true
		default:
			return fmt.Errorf("unrecognized option %s", o.String())
		}
	}

	if !allowWildcards {
		args = append(args, "-f")
	}

	// Windows has an upper limit on the length of command lines when executing a command via
	// the CreateProcess family of APIs (which is what Go uses; see syscall\exec_windows.go).
	// That limit is 32,768 characters.

	// In the simple case of a single file whose path isn't CRAZY long, then just pass it on
	// the command line...

	if len(paths) == 1 && len(paths[0]) < 30000 {
		return p.sh.Cmdf(`%s add %s "%s"`, p.cmd(), strings.Join(args, " "), paths[0]).RunErr()
	}

	// ...For all other cases, use a temp file to hold the file path(s).

	fnCleanup, filename, err := WriteTempFile("p4harmonize_add_*.txt", strings.Join(paths, "\n"))
	if err != nil {
		return err
	}
	defer fnCleanup()

	return p.sh.Cmdf(`%s -x "%s" add %s`, p.cmd(), filename, strings.Join(args, " ")).RunErr()
}
