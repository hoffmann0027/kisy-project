package testdb

import "testing"

// TestMissingURLIsFatalOnlyInCI pins audit D-12: an unset TEST_DATABASE_URL
// must skip on a laptop and fail in CI. A skip in CI means the integration
// suite reported success without running a query — the state the September
// audit found, with every HIGH finding sitting in "tested" code.
func TestMissingURLIsFatalOnlyInCI(t *testing.T) {
	cases := map[string]bool{
		"":      false, // developer machine
		"0":     false, // explicitly not CI
		"false": false,
		"true":  true, // GitHub Actions
		"1":     true,
		" TRUE": true, // set with stray whitespace or another case
	}

	for ci, want := range cases {
		if got := missingURLIsFatal(ci); got != want {
			t.Errorf("missingURLIsFatal(%q) = %v, want %v", ci, got, want)
		}
	}
}
