// Package update provides self-update functionality: version comparison,
// update checking, and atomic binary replacement.
package update

import (
	"fmt"
	"strconv"
	"strings"
)

// Version is a parsed semantic version.
type Version struct {
	Major      int
	Minor      int
	Patch      int
	PreRelease []string // e.g., ["rc", "1"]
}

// Parse parses a semantic version string (e.g., "0.4.0", "1.2.3-rc1").
func Parse(s string) (Version, error) {
	s = strings.TrimLeft(s, "vV")
	if s == "" {
		return Version{}, fmt.Errorf("empty version string")
	}

	parts := strings.SplitN(s, "-", 2)
	nums := strings.Split(parts[0], ".")
	if len(nums) != 3 {
		return Version{}, fmt.Errorf("invalid semver: %q", s)
	}

	major, err := strconv.Atoi(nums[0])
	if err != nil || major < 0 {
		return Version{}, fmt.Errorf("invalid major version %q", nums[0])
	}
	minor, err := strconv.Atoi(nums[1])
	if err != nil || minor < 0 {
		return Version{}, fmt.Errorf("invalid minor version %q", nums[1])
	}
	patch, err := strconv.Atoi(nums[2])
	if err != nil || patch < 0 {
		return Version{}, fmt.Errorf("invalid patch version %q", nums[2])
	}

	v := Version{Major: major, Minor: minor, Patch: patch}
	if len(parts) > 1 && parts[1] != "" {
		v.PreRelease = strings.Split(parts[1], ".")
	}
	return v, nil
}

// MustParse parses a version and panics on error. Use for constants.
func MustParse(s string) Version {
	v, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return v
}

// Compare returns -1 if v < other, 0 if v == other, 1 if v > other.
// Pre-release versions sort before their release counterpart
// (0.5.0-rc1 < 0.5.0) and after the prior release (0.4.0 < 0.5.0-rc1).
func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		return cmp(v.Major, other.Major)
	}
	if v.Minor != other.Minor {
		return cmp(v.Minor, other.Minor)
	}
	if v.Patch != other.Patch {
		return cmp(v.Patch, other.Patch)
	}
	// Same release version — compare pre-release.
	if len(v.PreRelease) == 0 && len(other.PreRelease) > 0 {
		return 1 // release > pre-release
	}
	if len(v.PreRelease) > 0 && len(other.PreRelease) == 0 {
		return -1 // pre-release < release
	}
	return comparePreRelease(v.PreRelease, other.PreRelease)
}

func (v Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if len(v.PreRelease) > 0 {
		s += "-" + strings.Join(v.PreRelease, ".")
	}
	return s
}

func cmp(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func comparePreRelease(a, b []string) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		ai, errA := strconv.Atoi(a[i])
		bi, errB := strconv.Atoi(b[i])
		if errA == nil && errB == nil {
			if c := cmp(ai, bi); c != 0 {
				return c
			}
		} else if errA == nil && errB != nil {
			return -1 // numeric < non-numeric per semver spec
		} else if errA != nil && errB == nil {
			return 1  // non-numeric > numeric per semver spec
		} else {
			if a[i] < b[i] {
				return -1
			}
			if a[i] > b[i] {
				return 1
			}
		}
	}
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return 0
}
