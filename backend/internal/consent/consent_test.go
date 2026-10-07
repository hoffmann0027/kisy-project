package consent

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func ptr(s string) *string { return &s }

func TestRequiredUntilBothCurrentVersionsAreAccepted(t *testing.T) {
	cases := []struct {
		name    string
		privacy *string
		rules   *string
		want    bool
	}{
		{"never accepted anything", nil, nil, true},
		{"only the privacy policy", ptr(PrivacyVersion), nil, true},
		{"only the rules", nil, ptr(RulesVersion), true},
		{"the rules changed since", ptr(PrivacyVersion), ptr("2020-01-01"), true},
		{"the policy changed since", ptr("2020-01-01"), ptr(RulesVersion), true},
		{"both current", ptr(PrivacyVersion), ptr(RulesVersion), false},
	}
	for _, tc := range cases {
		if got := Required(tc.privacy, tc.rules); got != tc.want {
			t.Errorf("%s: Required = %v, want %v", tc.name, got, tc.want)
		}
	}

	if !Current().IsCurrent() {
		t.Fatal("Current() is not current")
	}
	if (Acceptance{}).IsCurrent() {
		t.Fatal("an empty acceptance counts as consent")
	}
}

// TestVersionsMatchTheTextsPeopleAreShown ties the two halves together. The
// texts and their versions live in the frontend; the server refuses any
// version but its own. If someone bumps one side and forgets the other, every
// sign-up fails with CONSENT_REQUIRED — or worse, a changed text goes out
// without anyone being asked to accept it again. Either way it should fail
// here, not in production.
func TestVersionsMatchTheTextsPeopleAreShown(t *testing.T) {
	path := filepath.Join("..", "..", "..", "frontend", "src", "shared", "config", "legalVersions.ts")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	for name, want := range map[string]string{"PRIVACY_VERSION": PrivacyVersion, "RULES_VERSION": RulesVersion} {
		m := regexp.MustCompile(`export const ` + name + ` = "([^"]+)"`).FindSubmatch(raw)
		if m == nil {
			t.Fatalf("%s is not declared in %s", name, path)
		}
		if got := string(m[1]); got != want {
			t.Errorf("%s: frontend says %q, backend says %q — change both together", name, got, want)
		}
	}
}
