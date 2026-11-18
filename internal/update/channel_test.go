package update

import (
	"context"
	"testing"
)

func TestStableManifestComesFromTheRepoTheBuildNames(t *testing.T) {
	was := repo
	t.Cleanup(func() { repo = was })
	repo = "https://github.com/someone/fork"

	got, err := Stable.Manifest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	const want = "https://github.com/someone/fork/releases/latest/download/manifest.json"
	if got != want {
		t.Errorf("the stable manifest is fetched from %s, want %s", got, want)
	}
}

func TestABuildThatNamesNoRepoFollowsUpstream(t *testing.T) {
	if want := "https://github.com/ygelfand/echolocal"; repo != want {
		t.Errorf("updates default to %s, want %s", repo, want)
	}
}
