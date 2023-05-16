package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/danbrakeley/p4harmonize/internal/p4"
)

type Expected struct {
	Match        string
	SrcOnly      string
	DstOnly      string
	CaseMismatch string
}

func Test_ReconcileNoCaseProblems(t *testing.T) {
	var cases = []struct {
		Name     string
		Src      string
		Dst      string
		expected Expected
	}{
		{"simple match (no digest)", "foo", "foo", Expected{"foo:foo", "", "", ""}},
		{"missing src", "", "foo", Expected{"", "", "foo", ""}},
		{"missing dst", "foo", "", Expected{"", "foo", "", ""}},
		{"complex match", "a,b,c,f", "b,c,d,f,g", Expected{"b:b,c:c,f:f", "a", "d,g", ""}},
		{"same digest", "foo+somedigest", "foo+somedigest", Expected{"", "", "", ""}},
		{"differing digest", "foo+somedigest", "foo+otherdigest", Expected{"foo:foo", "", "", ""}},
	}

	for _, tc := range cases {
		t.Run(tc.Name+" case sensitive", func(t *testing.T) {
			src, dst := makeDepotFilesFromString(tc.Src), makeDepotFilesFromString(tc.Dst)
			actual := Reconcile(src, dst)
			checkReconcileWithExpected(t, actual, tc.expected)
		})
		t.Run(tc.Name+" case insensitive", func(t *testing.T) {
			src, dst := makeDepotFilesFromString(tc.Src), makeDepotFilesFromString(tc.Dst)
			actual := Reconcile(src, dst, DstIsCaseInsensitive)
			checkReconcileWithExpected(t, actual, tc.expected)
		})
	}
}

func Test_ReconcileCaseProblems(t *testing.T) {
	var cases = []struct {
		Name   string
		Src    string
		Dst    string
		sens   Expected
		insens Expected
	}{
		{"case 1", "a,b,c", "A,b,", Expected{"a:A,b:b", "c", "", ""}, Expected{"b:b", "c", "", "a:A"}},
		{"case 2", "A,b,c", "a,B,", Expected{"A:a,b:B", "c", "", ""}, Expected{"", "c", "", "A:a,b:B"}},
		{"case 3", "a,b", "a,B,C", Expected{"a:a,b:B", "", "C", ""}, Expected{"a:a", "", "C", "b:B"}},
		{"case 4", "aaron,Beto,clarinet", "aaron,beta,clarineT",
			Expected{"aaron:aaron,clarinet:clarineT", "Beto", "beta", ""},
			Expected{"aaron:aaron", "Beto", "beta", "clarinet:clarineT"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name+" case sensitive", func(t *testing.T) {
			src, dst := makeDepotFilesFromString(tc.Src), makeDepotFilesFromString(tc.Dst)
			actual := Reconcile(src, dst)
			checkReconcileWithExpected(t, actual, tc.sens)
		})
		t.Run(tc.Name+" case insensitive", func(t *testing.T) {
			src, dst := makeDepotFilesFromString(tc.Src), makeDepotFilesFromString(tc.Dst)
			actual := Reconcile(src, dst, DstIsCaseInsensitive)
			checkReconcileWithExpected(t, actual, tc.insens)
		})
	}
}

func Test_ReconcileHasDifference(t *testing.T) {
	var cases = []struct {
		Name     string
		Src      string
		Dst      string
		Expected bool
	}{
		{"simple match (no digest)", "foo", "foo", true},
		{"missing src", "", "foo", true},
		{"missing dst", "foo", "", true},
		{"complex match", "a,b,c,f", "b,c,d,f,g", true},
		{"mismatched case", "a,b,c", "a,B,", true},
		{"same digest", "foo+somedigest", "foo+somedigest", false},
		{"differing digest", "foo+somedigest", "foo+otherdigest", true},
		{"mismatched case", "a,b", "A,b,", true},
	}

	for _, tc := range cases {
		t.Run(tc.Name+" case sensitive", func(t *testing.T) {
			src, dst := makeDepotFilesFromString(tc.Src), makeDepotFilesFromString(tc.Dst)
			actual := Reconcile(src, dst)

			hasDifference := actual.HasDifference()
			if tc.Expected != hasDifference {
				t.Errorf("expected HasDifference() to be %v, got %v", tc.Expected, hasDifference)
			}
		})
		t.Run(tc.Name+" case insensitive", func(t *testing.T) {
			src, dst := makeDepotFilesFromString(tc.Src), makeDepotFilesFromString(tc.Dst)
			actual := Reconcile(src, dst, DstIsCaseInsensitive)

			hasDifference := actual.HasDifference()
			if tc.Expected != hasDifference {
				t.Errorf("expected HasDifference() to be %v, got %v", tc.Expected, hasDifference)
			}
		})
	}
}

// helpers

func makeDepotFilesFromString(paths string) (depotFiles []p4.DepotFile) {
	for _, path := range strings.Split(paths, ",") {
		if len(path) == 0 {
			continue
		}

		pathAndDigest := strings.SplitN(path, "+", 2)
		if len(pathAndDigest) == 2 {
			depotFiles = append(depotFiles, p4.DepotFile{Path: pathAndDigest[0], Digest: pathAndDigest[1]})
		} else {
			depotFiles = append(depotFiles, p4.DepotFile{Path: path})
		}
	}
	return depotFiles
}

func checkReconcileWithExpected(t *testing.T, actual DepotFileDiff, expected Expected) {
	t.Helper()

	var actualMatch string
	if len(actual.Match) > 0 {
		for _, v := range actual.Match {
			actualMatch += fmt.Sprintf("%s:%s,", v[0].Path, v[1].Path)
		}
		actualMatch = actualMatch[:len(actualMatch)-1]
	}
	if actualMatch != expected.Match {
		t.Errorf("expected %s, got %s", expected.Match, actualMatch)
	}

	var actualSrcOnly string
	if len(actual.SrcOnly) > 0 {
		for _, v := range actual.SrcOnly {
			actualSrcOnly += v.Path + ","
		}
		actualSrcOnly = actualSrcOnly[:len(actualSrcOnly)-1]
	}
	if actualSrcOnly != expected.SrcOnly {
		t.Errorf("expected %s, got %s", expected.SrcOnly, actualSrcOnly)
	}

	var actualDstOnly string
	if len(actual.DstOnly) > 0 {
		for _, v := range actual.DstOnly {
			actualDstOnly += v.Path + ","
		}
		actualDstOnly = actualDstOnly[:len(actualDstOnly)-1]
	}
	if actualDstOnly != expected.DstOnly {
		t.Errorf("expected %s, got %s", expected.DstOnly, actualDstOnly)
	}

	var actualCaseMismatch string
	if len(actual.CaseMismatch) > 0 {
		for _, v := range actual.CaseMismatch {
			actualCaseMismatch += fmt.Sprintf("%s:%s,", v[0].Path, v[1].Path)
		}
		actualCaseMismatch = actualCaseMismatch[:len(actualCaseMismatch)-1]
	}
	if actualCaseMismatch != expected.CaseMismatch {
		t.Errorf("expected %s, got %s", expected.CaseMismatch, actualCaseMismatch)
	}
}
