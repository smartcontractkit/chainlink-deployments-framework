package artifacts

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_validateAbsoluteHTTPURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		raw     string
		wantErr string
	}{
		{name: "https", raw: "https://example.com/x.wasm", wantErr: ""},
		{name: "http", raw: "http://127.0.0.1:9/foo", wantErr: ""},
		{name: "http_with_port", raw: "http://example.com:8080/", wantErr: ""},
		{name: "relative_path", raw: "artifacts/foo.wasm", wantErr: "host"},
		{name: "no_scheme", raw: "example.com/foo", wantErr: "host"},
		{name: "ftp", raw: "ftp://example.com/a", wantErr: "scheme"},
		{name: "empty_host", raw: "http:///path", wantErr: "host"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validURL(tt.raw)
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestBinarySource_IsLocal_IsExternal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		src   BinarySource
		local bool
		ext   bool
	}{
		{"empty", BinarySource{}, false, false},
		{"local_only", NewBinarySourceLocal("/x/y.wasm"), true, false},
		{"external_only", BinarySource{ExternalRef: &externalBinaryRef{}}, false, true},
		{"both_flags", BinarySource{LocalPath: "/a", ExternalRef: &externalBinaryRef{}}, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.local, tt.src.IsLocal())
			require.Equal(t, tt.ext, tt.src.IsExternal())
		})
	}
}

func TestBinarySource_Validate(t *testing.T) {
	t.Parallel()
	validSHA := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" // empty sha256
	tests := []struct {
		name    string
		src     BinarySource
		wantErr string
	}{
		{
			name: "ok_local",
			src: BinarySource{
				LocalPath: t.TempDir() + "/dummy.wasm", // path format only; ArtifactsResolver.ResolveBinary would stat
			},
		},
		{
			name: "ok_external_url",
			src: BinarySource{
				ExternalRef: &externalBinaryRef{
					URL:    "https://example.com/x.wasm",
					SHA256: validSHA,
				},
			},
		},
		{
			name: "ok_external_release",
			src: BinarySource{
				ExternalRef: &externalBinaryRef{
					Repo:       "org/repo",
					ReleaseTag: "v1",
					AssetName:  "binary.wasm",
					SHA256:     validSHA,
				},
			},
		},
		{
			name:    "missing_both",
			src:     BinarySource{},
			wantErr: "localPath or externalRef is required",
		},
		{
			name: "both_local_and_external",
			src: BinarySource{
				LocalPath:   "/a.wasm",
				ExternalRef: &externalBinaryRef{URL: "https://x", SHA256: validSHA},
			},
			wantErr: "either localPath or externalRef, not both",
		},
		{
			name: "external_nil",
			src: BinarySource{
				ExternalRef: nil,
				LocalPath:   "",
			},
			wantErr: "localPath or externalRef is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.src.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func Test_externalBinaryRef_validate(t *testing.T) {
	t.Parallel()
	validSHA := "abababababababababababababababababababababababababababababababab"
	tests := []struct {
		name    string
		ref     *externalBinaryRef
		wantErr string
	}{
		{
			name: "url_ok",
			ref: &externalBinaryRef{
				URL:    "https://example.com/a.wasm",
				SHA256: validSHA,
			},
		},
		{
			name: "release_ok",
			ref: &externalBinaryRef{
				Repo: "o/r", ReleaseTag: "t", AssetName: "a.wasm", SHA256: validSHA,
			},
		},
		{
			name: "url_and_release",
			ref: &externalBinaryRef{
				URL: "https://x", Repo: "o/r", ReleaseTag: "t", AssetName: "a", SHA256: validSHA,
			},
			wantErr: "either url or repo/releaseTag/assetName, not both",
		},
		{
			name:    "missing_mode",
			ref:     &externalBinaryRef{SHA256: validSHA},
			wantErr: "url or repo/releaseTag/assetName is required",
		},
		{
			name:    "missing_sha",
			ref:     &externalBinaryRef{URL: "https://x"},
			wantErr: "sha256 is required",
		},
		{
			name: "bad_repo",
			ref: &externalBinaryRef{
				Repo: "nope", ReleaseTag: "t", AssetName: "a", SHA256: validSHA,
			},
			wantErr: "owner/name",
		},
		{
			name: "invalid_url_relative",
			ref: &externalBinaryRef{
				URL: "artifacts/foo.wasm", SHA256: validSHA,
			},
			wantErr: "host",
		},
		{
			name: "invalid_url_ftp",
			ref: &externalBinaryRef{
				URL: "ftp://example.com/a.wasm", SHA256: validSHA,
			},
			wantErr: "scheme",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.ref.validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func Test_externalBinaryRef_isURL_isGitHubRelease(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		ref     *externalBinaryRef
		wantURL bool
		wantRel bool
	}{
		{"https_url", &externalBinaryRef{URL: "https://example.com/a.wasm"}, true, false},
		{"release", &externalBinaryRef{Repo: "a/b", ReleaseTag: "v", AssetName: "x"}, false, true},
		{"empty", &externalBinaryRef{}, false, false},
		// isURL is only "url field set"; validate rejects invalid URLs.
		{"url_field_not_valid_http", &externalBinaryRef{URL: "totally-not-a-url"}, true, false},
		{"url_field_relative_looking", &externalBinaryRef{URL: "artifacts/foo.wasm"}, true, false},
		{"url_whitespace_only", &externalBinaryRef{URL: "   \t "}, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.wantURL, tt.ref.isURL(), "isURL should reflect non-empty url field only")
			require.Equal(t, tt.wantRel, tt.ref.isGitHubRelease())
		})
	}
}

func TestConfigSource_IsLocal_IsExternal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		src   ConfigSource
		local bool
		ext   bool
	}{
		{"empty", ConfigSource{}, false, false},
		{"local", NewConfigSourceLocal("/c.json"), true, false},
		{"external", ConfigSource{ExternalRef: &externalConfigRef{}}, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.local, tt.src.IsLocal())
			require.Equal(t, tt.ext, tt.src.IsExternal())
		})
	}
}

func TestConfigSource_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		src     ConfigSource
		wantErr string
	}{
		{
			name: "ok_local",
			src:  ConfigSource{LocalPath: "/tmp/cfg.json"},
		},
		{
			name: "ok_url",
			src: ConfigSource{
				ExternalRef: &externalConfigRef{URL: "https://example.com/c.json"},
			},
		},
		{
			name: "ok_github",
			src: ConfigSource{
				ExternalRef: &externalConfigRef{Repo: "o/r", Ref: "main", Path: "cfg.json"},
			},
		},
		{
			name:    "missing",
			src:     ConfigSource{},
			wantErr: "localPath or externalRef is required",
		},
		{
			name: "both",
			src: ConfigSource{
				LocalPath:   "/a.json",
				ExternalRef: &externalConfigRef{URL: "https://x"},
			},
			wantErr: "either localPath or externalRef, not both",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.src.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func Test_externalConfigRef_validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		ref     *externalConfigRef
		wantErr string
	}{
		{name: "url_ok", ref: &externalConfigRef{URL: "https://x"}},
		{name: "gh_ok", ref: &externalConfigRef{Repo: "o/r", Ref: "r", Path: "p"}},
		{name: "gh_ok_leading_slashes", ref: &externalConfigRef{Repo: "o/r", Ref: "r", Path: "///p.json"}},
		{
			name:    "both",
			ref:     &externalConfigRef{URL: "https://x", Repo: "o/r", Ref: "r", Path: "p"},
			wantErr: "either url or repo/ref/path, not both",
		},
		{
			name:    "incomplete_gh",
			ref:     &externalConfigRef{Repo: "o/r", Ref: "r"},
			wantErr: "path must name a non-empty file path",
		},
		{
			name:    "gh_path_slash_only",
			ref:     &externalConfigRef{Repo: "o/r", Ref: "r", Path: "/"},
			wantErr: "path must name a non-empty file path",
		},
		{
			name:    "gh_path_slashes_only",
			ref:     &externalConfigRef{Repo: "o/r", Ref: "r", Path: "////"},
			wantErr: "path must name a non-empty file path",
		},
		{
			name:    "bad_repo",
			ref:     &externalConfigRef{Repo: "bad", Ref: "r", Path: "p"},
			wantErr: "owner/name",
		},
		{
			name:    "invalid_url_relative",
			ref:     &externalConfigRef{URL: "config.json"},
			wantErr: "host",
		},
		{
			name:    "invalid_url_ftp",
			ref:     &externalConfigRef{URL: "ftp://example.com/c.json"},
			wantErr: "scheme",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.ref.validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func Test_externalConfigRef_isURL_isGitHubFile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		ref    *externalConfigRef
		url    bool
		ghFile bool
	}{
		{"https_url", &externalConfigRef{URL: "https://example.com/c.json"}, true, false},
		{"gh", &externalConfigRef{Repo: "a/b", Ref: "r", Path: "p"}, false, true},
		{"gh_path_leading_slash", &externalConfigRef{Repo: "a/b", Ref: "r", Path: "/p"}, false, true},
		{"gh_path_only_slashes", &externalConfigRef{Repo: "a/b", Ref: "r", Path: "///"}, false, false},
		// isURL is only "url field set"; validate rejects invalid URLs.
		{"url_field_not_valid_http", &externalConfigRef{URL: "not-a-url"}, true, false},
		{"url_whitespace_only", &externalConfigRef{URL: " "}, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.url, tt.ref.isURL(), "isURL should reflect non-empty url field only")
			require.Equal(t, tt.ghFile, tt.ref.isGitHubFile())
		})
	}
}

func TestWorkflowBundle_Validate(t *testing.T) {
	t.Parallel()
	validSHA := "abababababababababababababababababababababababababababababababab"
	tests := []struct {
		name    string
		bundle  WorkflowBundle
		wantErr string
	}{
		{
			name: "ok",
			bundle: WorkflowBundle{
				WorkflowName: "wf",
				Binary: BinarySource{
					LocalPath: "/x.wasm",
				},
				Config: ConfigSource{
					LocalPath: "/c.json",
				},
			},
		},
		{
			name: "empty_name",
			bundle: WorkflowBundle{
				WorkflowName: "",
				Binary:       NewBinarySourceLocal("/x.wasm"),
				Config:       NewConfigSourceLocal("/c.json"),
			},
			wantErr: "workflowName is required",
		},
		{
			name: "bad_binary",
			bundle: WorkflowBundle{
				WorkflowName: "w",
				Binary:       BinarySource{},
				Config:       NewConfigSourceLocal("/c.json"),
			},
			wantErr: "binary",
		},
		{
			name: "bad_config",
			bundle: WorkflowBundle{
				WorkflowName: "w",
				Binary: BinarySource{
					ExternalRef: &externalBinaryRef{
						URL: "https://x", SHA256: validSHA,
					},
				},
				Config: ConfigSource{},
			},
			wantErr: "config",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.bundle.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}
