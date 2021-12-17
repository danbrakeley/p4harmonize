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
		{"simple match", "foo", "foo", "foo:foo", "", ""},
		{"missing src", "", "foo", "", "", "foo"},
		{"missing dst", "foo", "", "", "foo", ""},
		{"complex match", "a,b,c,f", "b,c,d,f,g", "b:b,c:c,f:f", "a", "d,g"},
		{"mismatched case", "a,b,c", "a,B,", "a:a,b:B", "c", ""},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			src, dst := makeDepotFilesFromStrings(tc.Src, tc.Dst)
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
		{"simple match", "foo", "foo", false},
		{"missing src", "", "foo", true},
		{"missing dst", "foo", "", true},
		{"complex match", "a,b,c,f", "b,c,d,f,g", true},
		{"mismatched case", "a,b,c", "a,B,", true},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			src, dst := makeDepotFilesFromStrings(tc.Src, tc.Dst)
			actual := Reconcile(src, dst)

			if tc.Expected != actual.HasDifference {
				t.Errorf("expected HasDifference to be %v, got %v", tc.Expected, actual.HasDifference)
			}
		})
	}
}

func makeDepotFilesFromStrings(s, d string) (src []p4.DepotFile, dst []p4.DepotFile) {
	for _, path := range strings.Split(s, ",") {
		if len(path) == 0 {
			continue
		}
		src = append(src, p4.DepotFile{Path: path})
	}
	for _, path := range strings.Split(d, ",") {
		if len(path) == 0 {
			continue
		}
		dst = append(dst, p4.DepotFile{Path: path})
	}
	return src, dst
}
