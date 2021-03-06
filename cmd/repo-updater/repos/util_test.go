package repos

import (
	"testing"

	log15 "gopkg.in/inconshreveable/log15.v2"
)

func TestSetUserinfoBestEffort(t *testing.T) {
	cases := []struct {
		rawurl   string
		username string
		password string
		want     string
	}{
		// no-op
		{"https://foo.com/foo/bar", "", "", "https://foo.com/foo/bar"},
		// invalid name is returned as is
		{":/foo.com/foo/bar", "u", "p", ":/foo.com/foo/bar"},

		// no user details in rawurl
		{"https://foo.com/foo/bar", "u", "p", "https://u:p@foo.com/foo/bar"},
		{"https://foo.com/foo/bar", "u", "", "https://u@foo.com/foo/bar"},
		{"https://foo.com/foo/bar", "", "p", "https://foo.com/foo/bar"},

		// user set already
		{"https://x@foo.com/foo/bar", "u", "p", "https://x:p@foo.com/foo/bar"},
		{"https://x@foo.com/foo/bar", "u", "", "https://x@foo.com/foo/bar"},
		{"https://x@foo.com/foo/bar", "", "p", "https://x:p@foo.com/foo/bar"},

		// user and password set already
		{"https://x:y@foo.com/foo/bar", "u", "p", "https://x:y@foo.com/foo/bar"},
		{"https://x:y@foo.com/foo/bar", "u", "", "https://x:y@foo.com/foo/bar"},
		{"https://x:y@foo.com/foo/bar", "", "p", "https://x:y@foo.com/foo/bar"},

		// empty password
		{"https://x:@foo.com/foo/bar", "u", "p", "https://x:@foo.com/foo/bar"},
		{"https://x:@foo.com/foo/bar", "u", "", "https://x:@foo.com/foo/bar"},
		{"https://x:@foo.com/foo/bar", "", "p", "https://x:@foo.com/foo/bar"},
	}
	for _, c := range cases {
		got := setUserinfoBestEffort(c.rawurl, c.username, c.password)
		if got != c.want {
			t.Errorf("setUserinfoBestEffort(%q, %q, %q): got %q want %q", c.rawurl, c.username, c.password, got, c.want)
		}
	}
}

func init() {
	if !testing.Verbose() {
		log15.Root().SetHandler(log15.DiscardHandler())
	}
}
