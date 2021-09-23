package p4

import (
	"testing"
)

func Test_Escape(t *testing.T) {
	var cases = []struct {
		Name      string
		Unescaped string
		Escaped   string
	}{
		{"no special characters", `c:\foo\Icon20.png`, `c:\foo\Icon20.png`},
		{"at", `@`, `%40`},
		{"hash", `#`, `%23`},
		{"asterisk", `*`, `%2A`},
		{"percent", `%`, `%25`},
		{"combination of special characters",
			`100% *** CL#385 frank@example.gov`,
			`100%25 %2A%2A%2A CL%23385 frank%40example.gov`,
		},
		{"actual error",
			`D:\Proletariat\p4harmonize\local\p4\src\Engine\Icon30@2x.png`,
			`D:\Proletariat\p4harmonize\local\p4\src\Engine\Icon30%402x.png`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			actual := EscapePath(tc.Unescaped)

			if actual != tc.Escaped {
				t.Errorf("Escape: Expected: `%s`, Actual: `%s`", tc.Escaped, actual)
			}

			actual, err := UnescapePath(tc.Escaped)
			if err != nil {
				t.Fatalf("error unescaping: %v", err)
			}

			if actual != tc.Unescaped {
				t.Errorf("Unescape: Expected: `%s`, Actual: `%s`", tc.Unescaped, actual)
			}
		})
	}
}

func Test_UnescapeErrors(t *testing.T) {
	var cases = []struct {
		Name    string
		Escaped string
	}{
		{"end too soon", `foo%2`},
		{"not a hex number", `%2G`},
		{"misuse of percent", `100%`},
		{"invalid percent escape", `200%%`},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			un, err := UnescapePath(tc.Escaped)
			if err == nil {
				t.Fatalf("expected error when parsing '%s'", un)
			}
		})
	}
}
