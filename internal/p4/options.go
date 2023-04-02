package p4

// Option is a sort of algebraic enum, which allows modifiers to be set for specific p4 functions.
// It is up to each function to support each option or not, as makes sense.

type Option interface {
	isOption()
	String() string
}

// Changelist holds a changelist number as an int64 (<=0 means use the default cl)

func Changelist(cl int64) oChangelist {
	return oChangelist{CL: cl}
}

type oChangelist struct {
	CL int64
}

func (_ oChangelist) isOption()      {}
func (_ oChangelist) String() string { return "Changelist" }

// Type holds a filetype as a string

func Type(t string) oType {
	return oType{Type: t}
}

type oType struct {
	Type string
}

func (_ oType) isOption()      {}
func (_ oType) String() string { return "Type" }

// Keep means to keep local files on disk (don't make local changes, just update the server)

var Keep oKeep

type oKeep struct{}

func (_ oKeep) isOption()      {}
func (_ oKeep) String() string { return "Keep" }

// Do not perform any ignore checking, i.e. ignore any settings specified by P4IGNORE.

var DoNotIgnore oDoNotIgnore

type oDoNotIgnore struct{}

func (_ oDoNotIgnore) isOption()      {}
func (_ oDoNotIgnore) String() string { return "DoNotIgnore" }

// Allow wildcards in file names (see [p4 add](https://www.perforce.com/manuals/cmdref/Content/CmdRef/p4_add.html))

var AllowWildcards oAllowWildcards

type oAllowWildcards struct{}

func (_ oAllowWildcards) isOption()      {}
func (_ oAllowWildcards) String() string { return "AllowWildcards" }
