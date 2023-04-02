package p4

// SubmitChangelist submits the given changelist
func (p *P4) SubmitChangelist(cl int64) error {
	return p.sh.Cmdf(`%s submit -c %d`, p.cmd(), cl).RunErr()
}
