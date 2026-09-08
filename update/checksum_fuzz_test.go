package update

import "testing"

func FuzzLookupChecksum(f *testing.F) {
	f.Add("abc  tool-linux-amd64\n", "tool-linux-amd64")
	f.Add("abc *tool.exe\r\n", "tool.exe")
	f.Add("", "x")
	f.Add("garbage line\n\n  \n", "x")
	f.Fuzz(func(t *testing.T, manifest, asset string) {
		sum, ok := lookupChecksum(manifest, asset)
		if ok && sum == "" {
			t.Fatalf("found empty checksum for %q in %q", asset, manifest)
		}
		if !ok && sum != "" {
			t.Fatalf("checksum without ok for %q in %q", asset, manifest)
		}
	})
}
