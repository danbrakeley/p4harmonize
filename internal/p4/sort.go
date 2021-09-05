package p4

import (
	"sort"
	"strings"
)

// CaseInsensitive allows sorting string slices in a case insesitive manner via sort.Sort(CaseInsesitive(x))
type CaseInsensitive []string

func (x CaseInsensitive) Len() int           { return len(x) }
func (x CaseInsensitive) Less(i, j int) bool { return strings.ToLower(x[i]) < strings.ToLower(x[j]) }
func (x CaseInsensitive) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }

// sortCaseInsensitive performs an in-place case-insensitive, descending sort of the given string slice
func sortCaseInsensitive(s []string) {
	sort.Sort(CaseInsensitive(s))
}
