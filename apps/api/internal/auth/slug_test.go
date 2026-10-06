package auth

import "testing"

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Depot Tirta Sejuk":        "depot-tirta-sejuk",
		"  Air   Segar!!  ":        "air-segar",
		"Depot_Bu Rina (Cabang 2)": "depot-bu-rina-cabang-2",
		"ÄÖÜ":                      "depot",
		"":                         "depot",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}
