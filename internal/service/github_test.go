package service

import "testing"

func TestIsGithubRepoURL(t *testing.T) {
	cases := []struct {
		url string
		ok  bool
	}{
		{"https://github.com/golang/go", true},
		{"https://github.com/golang/go/issues/1", false},
		{"https://example.com/a/b", false},
	}
	for _, c := range cases {
		if IsGithubRepoURL(c.url) != c.ok {
			t.Fatalf("url=%s expect=%v", c.url, c.ok)
		}
	}
}
