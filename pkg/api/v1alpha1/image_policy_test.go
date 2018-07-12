package v1alpha1

import "testing"

func TestImagePolicyMatchMapping(t *testing.T) {
	tcs := []struct {
		name  string
		in    string
		out   string
		noErr bool

		from string
		to   string
	}{
		{"identity", "any", "any", true, "", ""},

		{"map (from)", "v1.0.0", "1.0.0", true, "v{{.Tag}}", ""},
		{"map (suffix)", "v1.0.0-thing", "1.0.0", true, "v{{.Tag}}-thing", ""},
		{"map (to)", "1.0.0", "great-1.0.0-thing", true, "", "great-{{.Tag}}-thing"},
		{"map (both)", "v1.0.0", "marketplace-1.0.0", true, "v{{.Tag}}", "marketplace-{{.Tag}}"},

		{"no match in from", "1.0.0", "", false, "v{{.Tag}}", ""},

		{"error (from / no template)", "any", "", false, "v", ""},
		{"error (from / bad template)", "any", "", false, "{{.Tag", ""},
		{"error (to / no template)", "any", "", false, "", "v"},
		{"error (to / other template)", "any", "", false, "", "v{{.Food}}"},
		{"error (to / bad template)", "any", "", false, "", "{{.Tag"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			m := ImagePolicyMatchMapping{From: tc.from, To: tc.to}
			out, err := m.Map(tc.in)
			if tc.noErr && err != nil {
				t.Fatal("expected no err but got one:", err)
			}

			if !tc.noErr && err == nil {
				t.Fatal("expected err but got none.")
			}

			if out != tc.out {
				t.Error("wrong mapping. expected:", tc.out, "got:", out)
			}
		})
	}
}
