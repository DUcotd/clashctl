package mihomo

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestFetchLatestMihomoReleaseDoesNotUseMirrorByDefault(t *testing.T) {
	t.Setenv("CLASHCTL_ALLOW_UNTRUSTED_MIRROR", "")

	prevFetchJSON := fetchJSON
	fetchJSON = func(url string, timeout time.Duration, dest any) error {
		if strings.HasPrefix(url, "https://api.github.com/") {
			return fmt.Errorf("primary down")
		}
		t.Fatalf("fetchJSON() should not be called for mirror URL: %s", url)
		return nil
	}
	t.Cleanup(func() {
		fetchJSON = prevFetchJSON
	})

	_, err := fetchLatestMihomoRelease()
	if err == nil {
		t.Fatal("fetchLatestMihomoRelease() should fail when mirror fallback is disabled")
	}
}

func TestFetchLatestMihomoReleaseAllowsMirrorWithExplicitOptIn(t *testing.T) {
	t.Setenv("CLASHCTL_ALLOW_UNTRUSTED_MIRROR", "1")

	prevFetchJSON := fetchJSON
	fetchJSON = func(url string, timeout time.Duration, dest any) error {
		release, ok := dest.(*MihomoRelease)
		if !ok {
			t.Fatalf("dest type = %T, want *MihomoRelease", dest)
		}
		if strings.HasPrefix(url, "https://api.github.com/") {
			return fmt.Errorf("primary down")
		}
		release.TagName = "v1.2.3"
		return nil
	}
	t.Cleanup(func() {
		fetchJSON = prevFetchJSON
	})

	release, err := fetchLatestMihomoRelease()
	if err != nil {
		t.Fatalf("fetchLatestMihomoRelease() error = %v", err)
	}
	if release.TagName != "v1.2.3" {
		t.Fatalf("release.TagName = %q, want v1.2.3", release.TagName)
	}
}
