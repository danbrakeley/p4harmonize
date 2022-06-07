package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/proletariatgames/p4harmonize/internal/p4"
)

func Test_ReconcileJustPaths(t *testing.T) {
	var cases = []struct {
		Name    string
		Src     string
		Dst     string
		Match   string
		SrcOnly string
		DstOnly string
	}{
		{"simple match (no digest)", "foo", "foo", "foo:foo", "", ""},
		{"missing src", "", "foo", "", "", "foo"},
		{"missing dst", "foo", "", "", "foo", ""},
		{"complex match", "a,b,c,f", "b,c,d,f,g", "b:b,c:c,f:f", "a", "d,g"},
		{"same digest", "foo+somedigest", "foo+somedigest", "", "", ""},
		{"differing digest", "foo+somedigest", "foo+otherdigest", "foo:foo", "", ""},
		{"mismatched case", "a,b,c", "a,B,", "a:a,b:B", "c", ""},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			src, dst := makeDepotFilesFromString(tc.Src), makeDepotFilesFromString(tc.Dst)
			actual := Reconcile(src, dst)

			var actualMatch string
			if len(actual.Match) > 0 {
				for _, v := range actual.Match {
					actualMatch += fmt.Sprintf("%s:%s,", v[0].Path, v[1].Path)
				}
				actualMatch = actualMatch[:len(actualMatch)-1]
			}
			if tc.Match != actualMatch {
				t.Errorf("expected %s, got %s", tc.Match, actualMatch)
			}

			var actualSrcOnly string
			if len(actual.SrcOnly) > 0 {
				for _, v := range actual.SrcOnly {
					actualSrcOnly += v.Path + ","
				}
				actualSrcOnly = actualSrcOnly[:len(actualSrcOnly)-1]
			}
			if tc.SrcOnly != actualSrcOnly {
				t.Errorf("expected %s, got %s", tc.SrcOnly, actualSrcOnly)
			}

			var actualDstOnly string
			if len(actual.DstOnly) > 0 {
				for _, v := range actual.DstOnly {
					actualDstOnly += v.Path + ","
				}
				actualDstOnly = actualDstOnly[:len(actualDstOnly)-1]
			}
			if tc.DstOnly != actualDstOnly {
				t.Errorf("expected %s, got %s", tc.DstOnly, actualDstOnly)
			}
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
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			src, dst := makeDepotFilesFromString(tc.Src), makeDepotFilesFromString(tc.Dst)
			actual := Reconcile(src, dst)

			if tc.Expected != actual.HasDifference {
				t.Errorf("expected HasDifference to be %v, got %v", tc.Expected, actual.HasDifference)
			}
		})
	}
}

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
