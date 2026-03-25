package artifacts

import (
	"testing"

	"github.com/stretchr/testify/require"

	cfgenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
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
		{"local_only", BinarySource{LocalPath: "/x/y.wasm"}, true, false},
		{"external_only", BinarySource{ExternalRef: &ExternalBinaryRef{}}, false, true},
		{"both_flags", BinarySource{LocalPath: "/a", ExternalRef: &ExternalBinaryRef{}}, true, true},
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
				ExternalRef: &ExternalBinaryRef{
					URL:    "https://example.com/x.wasm",
					SHA256: validSHA,
				},
			},
		},
		{
			name: "ok_external_release",
			src: BinarySource{
				ExternalRef: &ExternalBinaryRef{
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
				ExternalRef: &ExternalBinaryRef{URL: "https://x", SHA256: validSHA},
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

func TestExternalBinaryRef_Validate(t *testing.T) {
	t.Parallel()
	validSHA := "abababababababababababababababababababababababababababababababab"
	tests := []struct {
		name    string
		ref     *ExternalBinaryRef
		wantErr string
	}{
		{name: "nil", ref: nil, wantErr: "nil"},
		{
			name: "url_ok",
			ref: &ExternalBinaryRef{
				URL:    "https://example.com/a.wasm",
				SHA256: validSHA,
			},
		},
		{
			name: "release_ok",
			ref: &ExternalBinaryRef{
				Repo: "o/r", ReleaseTag: "t", AssetName: "a.wasm", SHA256: validSHA,
			},
		},
		{
			name: "url_and_release",
			ref: &ExternalBinaryRef{
				URL: "https://x", Repo: "o/r", ReleaseTag: "t", AssetName: "a", SHA256: validSHA,
			},
			wantErr: "either url or repo/releaseTag/assetName, not both",
		},
		{
			name:    "missing_mode",
			ref:     &ExternalBinaryRef{SHA256: validSHA},
			wantErr: "url or repo/releaseTag/assetName is required",
		},
		{
			name:    "missing_sha",
			ref:     &ExternalBinaryRef{URL: "https://x"},
			wantErr: "sha256 is required",
		},
		{
			name: "bad_repo",
			ref: &ExternalBinaryRef{
				Repo: "nope", ReleaseTag: "t", AssetName: "a", SHA256: validSHA,
			},
			wantErr: "owner/name",
		},
		{
			name: "invalid_url_relative",
			ref: &ExternalBinaryRef{
				URL: "artifacts/foo.wasm", SHA256: validSHA,
			},
			wantErr: "host",
		},
		{
			name: "invalid_url_ftp",
			ref: &ExternalBinaryRef{
				URL: "ftp://example.com/a.wasm", SHA256: validSHA,
			},
			wantErr: "scheme",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.ref.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestExternalBinaryRef_IsURL_IsGitHubRelease(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		ref     *ExternalBinaryRef
		wantURL bool
		wantRel bool
	}{
		{"nil", nil, false, false},
		{"https_url", &ExternalBinaryRef{URL: "https://example.com/a.wasm"}, true, false},
		{"release", &ExternalBinaryRef{Repo: "a/b", ReleaseTag: "v", AssetName: "x"}, false, true},
		{"empty", &ExternalBinaryRef{}, false, false},
		// IsURL is only "url field set"; [ExternalBinaryRef.Validate] rejects these with [validURL].
		{"url_field_not_valid_http", &ExternalBinaryRef{URL: "totally-not-a-url"}, true, false},
		{"url_field_relative_looking", &ExternalBinaryRef{URL: "artifacts/foo.wasm"}, true, false},
		{"url_whitespace_only", &ExternalBinaryRef{URL: "   \t "}, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.wantURL, tt.ref.IsURL(), "IsURL should reflect non-empty url field only")
			require.Equal(t, tt.wantRel, tt.ref.IsGitHubRelease())
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
		{"local", ConfigSource{LocalPath: "/c.json"}, true, false},
		{"external", ConfigSource{ExternalRef: &ExternalConfigRef{}}, false, true},
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
				ExternalRef: &ExternalConfigRef{URL: "https://example.com/c.json"},
			},
		},
		{
			name: "ok_github",
			src: ConfigSource{
				ExternalRef: &ExternalConfigRef{Repo: "o/r", Ref: "main", Path: "cfg.json"},
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
				ExternalRef: &ExternalConfigRef{URL: "https://x"},
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

func TestExternalConfigRef_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		ref     *ExternalConfigRef
		wantErr string
	}{
		{name: "nil", ref: nil, wantErr: "nil"},
		{name: "url_ok", ref: &ExternalConfigRef{URL: "https://x"}},
		{name: "gh_ok", ref: &ExternalConfigRef{Repo: "o/r", Ref: "r", Path: "p"}},
		{name: "gh_ok_leading_slashes", ref: &ExternalConfigRef{Repo: "o/r", Ref: "r", Path: "///p.json"}},
		{
			name:    "both",
			ref:     &ExternalConfigRef{URL: "https://x", Repo: "o/r", Ref: "r", Path: "p"},
			wantErr: "either url or repo/ref/path, not both",
		},
		{
			name:    "incomplete_gh",
			ref:     &ExternalConfigRef{Repo: "o/r", Ref: "r"},
			wantErr: "path must name a non-empty file path",
		},
		{
			name:    "gh_path_slash_only",
			ref:     &ExternalConfigRef{Repo: "o/r", Ref: "r", Path: "/"},
			wantErr: "path must name a non-empty file path",
		},
		{
			name:    "gh_path_slashes_only",
			ref:     &ExternalConfigRef{Repo: "o/r", Ref: "r", Path: "////"},
			wantErr: "path must name a non-empty file path",
		},
		{
			name:    "bad_repo",
			ref:     &ExternalConfigRef{Repo: "bad", Ref: "r", Path: "p"},
			wantErr: "owner/name",
		},
		{
			name:    "invalid_url_relative",
			ref:     &ExternalConfigRef{URL: "config.json"},
			wantErr: "host",
		},
		{
			name:    "invalid_url_ftp",
			ref:     &ExternalConfigRef{URL: "ftp://example.com/c.json"},
			wantErr: "scheme",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.ref.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestExternalConfigRef_IsURL_IsGitHubFile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		ref    *ExternalConfigRef
		url    bool
		ghFile bool
	}{
		{"nil", nil, false, false},
		{"https_url", &ExternalConfigRef{URL: "https://example.com/c.json"}, true, false},
		{"gh", &ExternalConfigRef{Repo: "a/b", Ref: "r", Path: "p"}, false, true},
		{"gh_path_leading_slash", &ExternalConfigRef{Repo: "a/b", Ref: "r", Path: "/p"}, false, true},
		{"gh_path_only_slashes", &ExternalConfigRef{Repo: "a/b", Ref: "r", Path: "///"}, false, false},
		// IsURL is only "url field set"; [ExternalConfigRef.Validate] rejects these with [validURL].
		{"url_field_not_valid_http", &ExternalConfigRef{URL: "not-a-url"}, true, false},
		{"url_whitespace_only", &ExternalConfigRef{URL: " "}, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.url, tt.ref.IsURL(), "IsURL should reflect non-empty url field only")
			require.Equal(t, tt.ghFile, tt.ref.IsGitHubFile())
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
				Binary:       BinarySource{LocalPath: "/x.wasm"},
				Config:       ConfigSource{LocalPath: "/c.json"},
			},
			wantErr: "workflowName is required",
		},
		{
			name: "bad_binary",
			bundle: WorkflowBundle{
				WorkflowName: "w",
				Binary:       BinarySource{},
				Config:       ConfigSource{LocalPath: "/c.json"},
			},
			wantErr: "binary",
		},
		{
			name: "bad_config",
			bundle: WorkflowBundle{
				WorkflowName: "w",
				Binary: BinarySource{
					ExternalRef: &ExternalBinaryRef{
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

func TestWorkflowBundle_ApplyDeployDefaults(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		cre        cfgenv.CREConfig
		input      WorkflowBundle
		wantFamily string
	}{
		{
			name: "don_family_from_input",
			cre: cfgenv.CREConfig{
				DonFamily: "config-should-not-win",
			},
			input: WorkflowBundle{
				WorkflowName: "w",
				DonFamily:    "feeds-zone-a",
			},
			wantFamily: "feeds-zone-a",
		},
		{
			name: "don_family_from_loaded_cre_config",
			cre: cfgenv.CREConfig{
				DonFamily: "from-config",
			},
			input: WorkflowBundle{
				WorkflowName: "w",
			},
			wantFamily: "from-config",
		},
		{
			name: "don_family_empty_when_cre_empty",
			cre:  cfgenv.CREConfig{},
			input: WorkflowBundle{
				WorkflowName: "w",
			},
			wantFamily: "",
		},
		{
			name: "whitespace_bundle_then_cre_empty",
			cre:  cfgenv.CREConfig{},
			input: WorkflowBundle{
				WorkflowName: "w",
				DonFamily:    "   ",
			},
			wantFamily: "",
		},
		{
			name: "whitespace_bundle_then_cre_set",
			cre: cfgenv.CREConfig{
				DonFamily: "from-cre",
			},
			input: WorkflowBundle{
				WorkflowName: "w",
				DonFamily:    "   ",
			},
			wantFamily: "from-cre",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := tt.input
			w.ApplyDeployDefaults(tt.cre)
			require.Equal(t, tt.wantFamily, w.DonFamily)
		})
	}
}

func TestWorkflowBundle_ApplyDeployDefaults_nil(t *testing.T) {
	t.Parallel()
	var w *WorkflowBundle
	require.NotPanics(t, func() { w.ApplyDeployDefaults(cfgenv.CREConfig{}) })
}
