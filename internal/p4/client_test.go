package p4

import (
	"encoding/json"
	"testing"
)

func Test_ParseSpec(t *testing.T) {
	var cases = []struct {
		Name     string
		Spec     string
		Expected string
	}{
		{"one field",
			"... Client foo",
			`{"Client":"foo"}`,
		},
		{"one field and an empty line",
			"... Client foo\n",
			`{"Client":"foo"}`,
		},
		{"one field description",
			"... Description Something different\n\n",
			`{"Description":"Something different\n"}`,
		},
		{"two fields with description first",
			"... Description\nCreated by\n\n... Client foo\n",
			`{"Client":"foo","Description":"\nCreated by\n"}`,
		},
		{"two fields with description last",
			"... Client foo\n... Description\nCreated by\n\n",
			`{"Client":"foo","Description":"\nCreated by\n"}`,
		},
		{"three fields with description in middle",
			"... Client foo\n... Description\nCreated by\n\n... SubmitOptions submitunchanged",
			`{"Client":"foo","Description":"\nCreated by\n","SubmitOptions":"submitunchanged"}`,
		},
		{"no change to special characters in path",
			"... Root C:\\Mambo#5\\%23\\",
			`{"Root":"C:\\Mambo#5\\%23\\"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			parsed := ParseSpec(tc.Spec)
			b, err := json.Marshal(parsed)
			if err != nil {
				t.Fatalf("%v", err)
			}
			actual := string(b)

			if actual != tc.Expected {
				t.Fatalf("Expected:\n%s\nActual:\n%s", tc.Expected, actual)
			}
		})
	}
}
