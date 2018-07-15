package v1alpha1

import "testing"

func TestImagePolicyMatchConfig(t *testing.T) {
	tcs := []struct {
		name string

		hasName   bool
		hasLabels bool

		match *ImagePolicyMatch
	}{
		{"nil", true, false, nil},
		{"empty", true, false, &ImagePolicyMatch{}},

		{"both", true, true, &ImagePolicyMatch{
			Name: &ImagePolicyMatchMapping{},
			Labels: map[string]ImagePolicyMatchMapping{
				"org.fake.label": {},
			},
		}},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {

			n, l := tc.match.Config()

			if n != tc.hasName {
				t.Error("wrong value for name. expected:", tc.hasName, "got:", n)
			}

			if l != tc.hasLabels {
				t.Error("wrong value for labels. expected:", tc.hasLabels, "got:", l)
			}
		})
	}
}

func TestImagePolicyMatchMapName(t *testing.T) {
	tcs := []struct {
		name string

		in    string
		out   string
		noErr bool

		match *ImagePolicyMatch
	}{
		{"nil match is default", "tag", "tag", true, nil},
		{"zero value is default", "tag", "tag", true, &ImagePolicyMatch{}},

		{"err is propagated", "tag", "", false, &ImagePolicyMatch{
			Name: &ImagePolicyMatchMapping{From: "{{"},
		}},

		{"name is mapped", "tag", "vtag", true, &ImagePolicyMatch{
			Name: &ImagePolicyMatchMapping{To: "v{{.Tag}}"},
		}},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			out, err := tc.match.MapName(tc.in)

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

func TestImagePolicyMatchMatches(t *testing.T) {
	tcs := []struct {
		name string

		in       string
		tag      string
		labels   map[string]string
		expected bool
		noErr    bool

		match *ImagePolicyMatch
	}{
		{"nil match is default", "tag", "tag", nil, true, true, nil},
		{"zero value is default", "tag", "tag", nil, true, true, &ImagePolicyMatch{}},

		{"default no match", "tag", "not tag", nil, false, true, nil},

		{"err is propagated in name", "tag", "tag", nil, false, false, &ImagePolicyMatch{
			Name: &ImagePolicyMatchMapping{
				From: "{{",
			},
		}},

		{"match on labels", "tag", "not tag",
			map[string]string{
				"org.fake.label": "vtag",
			},
			true, true,
			&ImagePolicyMatch{
				Labels: map[string]ImagePolicyMatchMapping{
					"org.fake.label": {To: "v{{.Tag}}"},
				},
			},
		},

		{"exclude on labels", "tag", "not tag",
			map[string]string{
				"org.fake.label": "not tag",
			},
			false, true,
			&ImagePolicyMatch{
				Labels: map[string]ImagePolicyMatchMapping{
					"org.fake.label": {},
				},
			},
		},

		{"err is propagated in labels", "tag", "not tag",
			map[string]string{
				"org.fake.label": "tag",
			},
			false, false,
			&ImagePolicyMatch{
				Labels: map[string]ImagePolicyMatchMapping{
					"org.fake.label": {From: "{{."},
				},
			},
		},

		{"exclude on missing label", "tag", "tag",
			map[string]string{},
			false, true,
			&ImagePolicyMatch{
				Name: &ImagePolicyMatchMapping{},
				Labels: map[string]ImagePolicyMatchMapping{
					"org.fake.label": {},
				},
			},
		},

		{"match on all values", "tag", "tag",
			map[string]string{
				"org.fake.label":       "tag",
				"org.fake.other.label": "tasty",
			},
			true, true,
			&ImagePolicyMatch{
				Name: &ImagePolicyMatchMapping{},
				Labels: map[string]ImagePolicyMatchMapping{
					"org.fake.label":       {},
					"org.fake.other.label": {From: "{{.Tag}}g", To: "{{.Tag}}sty"},
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			out, err := tc.match.Matches(tc.in, tc.tag, tc.labels)
			if tc.noErr && err != nil {
				t.Fatal("expected no err but got one:", err)
			}

			if !tc.noErr && err == nil {
				t.Fatal("expected err but got none.")
			}

			if out != tc.expected {
				t.Error("wrong result. expected:", tc.expected, "got:", out)
			}
		})
	}
}

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
