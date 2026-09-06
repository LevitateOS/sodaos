package config

import "testing"

func TestBaseURL(t *testing.T) {
	for _, u := range []string{"https://soda.example.test", "http://127.0.0.1:3000"} {
		if err := BaseURL(u); err != nil {
			t.Fatal(err)
		}
	}
	for _, u := range []string{"file:///etc/passwd", "https://user:password@host", "https://host/path", "https://host?x=1"} {
		if BaseURL(u) == nil {
			t.Fatalf("accepted %q", u)
		}
	}
}
