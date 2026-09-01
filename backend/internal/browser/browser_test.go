package browser

import "testing"

func TestCommand(t *testing.T) {
	cases := []struct {
		goos     string
		wantName string
		wantArgs []string
	}{
		{"windows", "cmd", []string{"/c", "start", "", "http://x"}},
		{"darwin", "open", []string{"http://x"}},
		{"linux", "xdg-open", []string{"http://x"}},
	}
	for _, c := range cases {
		name, args := command(c.goos, "http://x")
		if name != c.wantName {
			t.Errorf("%s: name = %q, want %q", c.goos, name, c.wantName)
		}
		if len(args) != len(c.wantArgs) {
			t.Fatalf("%s: args = %v, want %v", c.goos, args, c.wantArgs)
		}
		for i := range args {
			if args[i] != c.wantArgs[i] {
				t.Errorf("%s: args[%d] = %q, want %q", c.goos, i, args[i], c.wantArgs[i])
			}
		}
	}
}
