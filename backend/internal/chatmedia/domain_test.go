package chatmedia

import (
	"reflect"
	"testing"
)

func TestExtractLinks(t *testing.T) {
	cases := []struct {
		name string
		text string
		want []string
	}{
		{"none", "просто текст без ссылок", nil},
		{"one", "смотри https://example.com/page", []string{"https://example.com/page"}},
		{
			"trailing punctuation trimmed",
			"вот (https://example.com/a), и https://example.com/b.",
			[]string{"https://example.com/a", "https://example.com/b"},
		},
		{
			"dedupe keeps order",
			"http://a.io и снова http://a.io и ещё https://b.io",
			[]string{"http://a.io", "https://b.io"},
		},
		{"query and fragment survive", "https://x.io/p?q=1&r=2#frag", []string{"https://x.io/p?q=1&r=2#frag"}},
		{"scheme required", "www.example.com не считается", nil},
	}
	for _, tc := range cases {
		if got := ExtractLinks(tc.text); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}
