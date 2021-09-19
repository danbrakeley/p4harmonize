package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/proletariatgames/p4harmonize/internal/p4"
)

func Test_Reconcile(t *testing.T) {
	var cases = []struct {
		Name      string
		Src       string
		Dst       string
		Match     string
		NearMatch string
		SrcOnly   string
		DstOnly   string
	}{
		{"simple match", "foo", "foo", "foo:foo", "", "", ""},
		{"missing src", "", "foo", "", "", "", "foo"},
		{"missing dst", "foo", "", "", "", "foo", ""},
		{"complex match", "a,b,c,f", "b,c,d,f,g", "b:b,c:c,f:f", "", "a", "d,g"},
		{"mismatched case", "a,b,c", "a,B,", "a:a", "b:B", "c", ""},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			var src, dst []p4.DepotFile
			for _, path := range strings.Split(tc.Src, ",") {
				if len(path) == 0 {
					continue
				}
				src = append(src, p4.DepotFile{Path: path})
			}
			for _, path := range strings.Split(tc.Dst, ",") {
				if len(path) == 0 {
					continue
				}
				dst = append(dst, p4.DepotFile{Path: path})
			}

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

			var actualNearMatch string
			if len(actual.NearMatch) > 0 {
				for _, v := range actual.NearMatch {
					actualNearMatch += fmt.Sprintf("%s:%s,", v[0].Path, v[1].Path)
				}
				actualNearMatch = actualNearMatch[:len(actualNearMatch)-1]
			}
			if tc.NearMatch != actualNearMatch {
				t.Errorf("expected %s, got %s", tc.NearMatch, actualNearMatch)
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
