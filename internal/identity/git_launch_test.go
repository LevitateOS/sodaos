package identity

import "testing"

func TestGitLaunchRejectsAuthorityAndMalformedParts(t *testing.T) {
	valid := GitLaunchRequest{CWD: "/workspace/repo", Remote: "origin", Owner: "soda-tester", Repository: "repo"}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"", ".", "..", "-option", "repo/path", "repo\\path", "repo\nheader", "repo\x00"} {
		request := valid
		request.Repository = part
		if request.Validate() == nil {
			t.Fatalf("accepted repository %q", part)
		}
	}
	valid.CWD = "relative"
	if valid.Validate() == nil {
		t.Fatal("accepted relative working directory")
	}
}
