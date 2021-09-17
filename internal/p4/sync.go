package p4

import "fmt"

// SyncLatest runs p4 sync ...#head
func (p *P4) SyncLatest() error {
	err := p.sh.Cmdf(`%s sync //%s/...#head`, p.cmd(), p.Client).RunErr()
	if err != nil {
		return fmt.Errorf(`error syncing %s to head: %w`, p.Client, err)
	}
	return nil
}

// SyncLatestNoDownload runs "p4 sync -k ...#head" which will:
// "Keep existing workspace files; update the have list without updating the client workspace"
func (p *P4) SyncLatestNoDownload() error {
	err := p.sh.Cmdf(`%s sync -k //%s/...#head`, p.cmd(), p.Client).Out(nil).RunErr()
	if err != nil {
		return fmt.Errorf(`error fake-syncing %s to head: %w`, p.Client, err)
	}
	return nil
}
