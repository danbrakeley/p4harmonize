package p4

import (
	"fmt"
	"strings"
)

// NeedsLogin determines if we have a valid ticket or not.
func (p *P4) NeedsLogin() (bool, error) {
	var sb strings.Builder
	sb.Grow(256)
	err := p.sh.Cmdf("%s login -s", p.cmd()).Out(nil).Err(&sb).RunErr()
	if err == nil {
		return false, nil
	}
	out := sb.String()
	if strings.HasPrefix(out, "Your session has expired, please login again") {
		return true, nil
	}
	return false, fmt.Errorf("%s", out)
}
