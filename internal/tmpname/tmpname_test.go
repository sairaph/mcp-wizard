package tmpname

import "testing"

func TestSuffix(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		s := Suffix(8)
		if len(s) != 8 {
			t.Fatalf("len = %d", len(s))
		}
		for _, c := range s {
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9') {
				t.Fatalf("unexpected character %q in %q", c, s)
			}
		}
		seen[s] = true
	}
	if len(seen) < 190 {
		t.Fatalf("suffixes are not random enough: %d distinct of 200", len(seen))
	}
	if Suffix(0) != "" {
		t.Fatal("Suffix(0) must be empty")
	}
}
