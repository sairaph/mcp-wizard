package update

import (
	"testing"
)

func TestParse_0_4_0(t *testing.T) {
	v, err := Parse("0.4.0")
	if err != nil {
		t.Fatal(err)
	}
	if v.Major != 0 || v.Minor != 4 || v.Patch != 0 || len(v.PreRelease) != 0 {
		t.Fatalf("unexpected version: %+v", v)
	}
}

func TestParse_VPrefix(t *testing.T) {
	v, err := Parse("v0.4.0")
	if err != nil {
		t.Fatal(err)
	}
	if v.Major != 0 || v.Minor != 4 || v.Patch != 0 {
		t.Fatalf("unexpected version: %+v", v)
	}
}

func TestParse_PreRelease(t *testing.T) {
	v, err := Parse("1.2.3-rc1")
	if err != nil {
		t.Fatal(err)
	}
	if v.Major != 1 || v.Minor != 2 || v.Patch != 3 {
		t.Fatalf("unexpected release: %+v", v)
	}
	if len(v.PreRelease) != 1 || v.PreRelease[0] != "rc1" {
		t.Fatalf("unexpected pre-release: %v", v.PreRelease)
	}
}

func TestParse_PreReleaseDotted(t *testing.T) {
	v, err := Parse("0.5.0-rc.1")
	if err != nil {
		t.Fatal(err)
	}
	if v.Major != 0 || v.Minor != 5 || v.Patch != 0 {
		t.Fatalf("unexpected release: %+v", v)
	}
	if len(v.PreRelease) != 2 || v.PreRelease[0] != "rc" || v.PreRelease[1] != "1" {
		t.Fatalf("unexpected pre-release: %v", v.PreRelease)
	}
}

func TestParse_Empty(t *testing.T) {
	_, err := Parse("")
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}

func TestParse_TwoParts(t *testing.T) {
	_, err := Parse("0.4")
	if err == nil {
		t.Fatal("expected error for 2-part version")
	}
}

func TestParse_Invalid(t *testing.T) {
	_, err := Parse("abc")
	if err == nil {
		t.Fatal("expected error for invalid version")
	}
}

func TestCompare_Equal(t *testing.T) {
	a := MustParse("1.2.3")
	b := MustParse("1.2.3")
	if c := a.Compare(b); c != 0 {
		t.Fatalf("expected 0, got %d", c)
	}
}

func TestCompare_Major(t *testing.T) {
	a := MustParse("2.0.0")
	b := MustParse("1.0.0")
	if c := a.Compare(b); c != 1 {
		t.Fatalf("expected 1, got %d", c)
	}
	if c := b.Compare(a); c != -1 {
		t.Fatalf("expected -1, got %d", c)
	}
}

func TestCompare_Minor(t *testing.T) {
	a := MustParse("1.3.0")
	b := MustParse("1.2.0")
	if c := a.Compare(b); c != 1 {
		t.Fatalf("expected 1, got %d", c)
	}
	if c := b.Compare(a); c != -1 {
		t.Fatalf("expected -1, got %d", c)
	}
}

func TestCompare_Patch(t *testing.T) {
	a := MustParse("1.2.5")
	b := MustParse("1.2.3")
	if c := a.Compare(b); c != 1 {
		t.Fatalf("expected 1, got %d", c)
	}
	if c := b.Compare(a); c != -1 {
		t.Fatalf("expected -1, got %d", c)
	}
}

func TestCompare_PreReleaseVsRelease(t *testing.T) {
	pre := MustParse("0.5.0-rc1")
	rel := MustParse("0.5.0")
	if c := pre.Compare(rel); c != -1 {
		t.Fatalf("expected -1 (pre < release), got %d", c)
	}
	if c := rel.Compare(pre); c != 1 {
		t.Fatalf("expected 1 (release > pre), got %d", c)
	}
}

func TestCompare_PreReleaseVsPrior(t *testing.T) {
	prior := MustParse("0.4.0")
	pre := MustParse("0.5.0-rc1")
	if c := prior.Compare(pre); c != -1 {
		t.Fatalf("expected -1 (prior < pre), got %d", c)
	}
	if c := pre.Compare(prior); c != 1 {
		t.Fatalf("expected 1 (pre > prior), got %d", c)
	}
}

func TestString_RoundTrip(t *testing.T) {
	cases := []string{"0.4.0", "1.2.3-rc1", "0.5.0-rc.1", "2.0.0"}
	for _, s := range cases {
		v, err := Parse(s)
		if err != nil {
			t.Fatal(err)
		}
		if got := v.String(); got != s {
			t.Fatalf("String() round-trip: %q != %q", got, s)
		}
	}
}

func TestMustParse_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustParse("invalid")
}

func FuzzParse(f *testing.F) {
	for _, s := range []string{"1.2.3", "v0.1.0", "1.2.3-rc.1+build", "", "1.2", "a.b.c", "1.2.3-", "01.2.3", "1.2.3+", "-1.2.3"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		v, err := Parse(in)
		if err != nil {
			return
		}
		// A parsed version must round-trip through String and compare equal.
		again, err := Parse(v.String())
		if err != nil {
			t.Fatalf("String() of %q = %q does not parse: %v", in, v.String(), err)
		}
		if again.Compare(v) != 0 {
			t.Fatalf("round trip changed ordering: %q -> %q", in, v.String())
		}
		if v.Major < 0 || v.Minor < 0 || v.Patch < 0 {
			t.Fatalf("negative component from %q", in)
		}
	})
}
