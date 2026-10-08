package clientversions

import "testing"

func TestParse(t *testing.T) {
	for in, want := range map[string]struct {
		version string
		build   int64
	}{
		"1.4.0 (29312345)":          {"1.4.0", 29312345},
		"dev-80c3ae3abc (29350000)": {"dev-80c3ae3abc", 29350000},
	} {
		v, b, ok := Parse(in)
		if !ok || v != want.version || b != want.build {
			t.Fatalf("%q: %q %d %v", in, v, b, ok)
		}
	}
	for _, bad := range []string{"", "native", "1.4.0", "1.4.0 (x)", "1.4.0 (-1)", "<script> (1)", "1.4.0  (1)"} {
		if _, _, ok := Parse(bad); ok {
			t.Fatalf("accepted %q", bad)
		}
	}
}
