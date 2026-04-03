package mihomo

import (
	"strings"
	"testing"
)

func TestGetGitHubMirrorURL_UsesURLJoinPath(t *testing.T) {
	tests := []struct {
		name    string
		mirror  string
		input   string
		wantHas string
	}{
		{
			name:    "custom mirror with trailing slash",
			mirror:  "https://mirror.example.com/",
			input:   "https://github.com/test/repo/releases/latest",
			wantHas: "mirror.example.com",
		},
		{
			name:    "custom mirror without trailing slash",
			mirror:  "https://mirror.example.com",
			input:   "https://api.github.com/repos/test/repo/releases/latest",
			wantHas: "mirror.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CLASHCTL_GITHUB_MIRROR", tt.mirror)
			got := GetGitHubMirrorURL(tt.input)
			if !strings.Contains(got, tt.wantHas) {
				t.Errorf("GetGitHubMirrorURL() = %v, want to contain %v", got, tt.wantHas)
			}
			if strings.Contains(got, "//") && !strings.HasPrefix(got, "https://") {
				t.Errorf("GetGitHubMirrorURL() has double slash: %v", got)
			}
		})
	}
}

func TestGetGitHubMirrorURL_NoMirror(t *testing.T) {
	t.Setenv("CLASHCTL_GITHUB_MIRROR", "")
	nonGitHub := "https://example.com/some/path"
	got := GetGitHubMirrorURL(nonGitHub)
	if got != nonGitHub {
		t.Errorf("GetGitHubMirrorURL() = %v, want %v", got, nonGitHub)
	}
}
