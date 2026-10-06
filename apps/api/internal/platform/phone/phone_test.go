package phone

import "testing"

func TestNormalize(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"081234567890", "6281234567890", true},
		{"0812-3456-7890", "6281234567890", true},
		{"+62 812 3456 7890", "6281234567890", true},
		{"6281234567890", "6281234567890", true},
		{"81234567890", "6281234567890", true},
		{"(0812) 3456 7890", "6281234567890", true},
		{"12345", "", false},
		{"+1 555 123 4567", "", false},
		{"", "", false},
		{"08123", "", false},
		{"628123456789012345", "", false},
	}
	for _, tc := range cases {
		got, err := Normalize(tc.in)
		if tc.ok && (err != nil || got != tc.want) {
			t.Errorf("Normalize(%q) = %q, %v; want %q", tc.in, got, err, tc.want)
		}
		if !tc.ok && err == nil {
			t.Errorf("Normalize(%q) should fail, got %q", tc.in, got)
		}
	}
}

func TestMask(t *testing.T) {
	if got := Mask("6281234567890"); got != "•••••••••7890" {
		t.Fatalf("got %q", got)
	}
	if got := Mask("123"); got != "123" {
		t.Fatalf("got %q", got)
	}
}
