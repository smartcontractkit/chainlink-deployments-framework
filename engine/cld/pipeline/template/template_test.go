package template

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	fresolvers "github.com/smartcontractkit/chainlink-deployments-framework/changeset/resolvers"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	cs "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
)

type templateStubChangeset struct{}

func (templateStubChangeset) Apply(_ fdeployment.Environment, _ any) (fdeployment.ChangesetOutput, error) {
	return fdeployment.ChangesetOutput{}, nil
}
func (templateStubChangeset) VerifyPreconditions(_ fdeployment.Environment, _ any) error {
	return nil
}

var _ fdeployment.ChangeSetV2[any] = (*templateStubChangeset)(nil)

type InputStruct struct {
	Chain string `yaml:"chain"`
	Value int    `yaml:"value"`
}

func inputTypeResolverForTest(in InputStruct) (any, error) {
	return in, nil
}

func TestGenerateMultiChangesetYAML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		domainName     string
		envKey         string
		changesetNames []string
		regSetup       func() *cs.ChangesetsRegistry
		rmSetup        func() *fresolvers.ConfigResolverManager
		wantErr        string
		checkGot       func(t *testing.T, got string)
	}{
		{
			name:           "no names nil",
			domainName:     "test",
			envKey:         "testnet",
			changesetNames: nil,
			regSetup:       func() *cs.ChangesetsRegistry { return cs.NewChangesetsRegistry() },
			rmSetup:        func() *fresolvers.ConfigResolverManager { return fresolvers.NewConfigResolverManager() },
			wantErr:        "no changeset names provided",
		},
		{
			name:           "no names empty",
			domainName:     "test",
			envKey:         "testnet",
			changesetNames: []string{},
			regSetup:       func() *cs.ChangesetsRegistry { return cs.NewChangesetsRegistry() },
			rmSetup:        func() *fresolvers.ConfigResolverManager { return fresolvers.NewConfigResolverManager() },
			wantErr:        "no changeset names provided",
		},
		{
			name:           "unknown changeset",
			domainName:     "test",
			envKey:         "testnet",
			changesetNames: []string{"0001_missing"},
			regSetup:       func() *cs.ChangesetsRegistry { return cs.NewChangesetsRegistry() },
			rmSetup:        func() *fresolvers.ConfigResolverManager { return fresolvers.NewConfigResolverManager() },
			wantErr:        "get configurations for changeset 0001_missing: changeset '0001_missing' not found",
		},
		{
			name:           "resolver not registered",
			domainName:     "test",
			envKey:         "testnet",
			changesetNames: []string{"0001_test"},
			regSetup: func() *cs.ChangesetsRegistry {
				resolver := func(m map[string]any) (any, error) { return m, nil }
				reg := cs.NewChangesetsRegistry()
				reg.Add("0001_test", cs.Configure(&templateStubChangeset{}).WithConfigResolver(resolver))

				return reg
			},
			rmSetup: func() *fresolvers.ConfigResolverManager { return fresolvers.NewConfigResolverManager() },
			wantErr: "generate section for changeset 0001_test: resolver for changeset 0001_test is not registered",
		},
		{
			name:           "with input type",
			domainName:     "mydomain",
			envKey:         "testnet",
			changesetNames: []string{"0001_test"},
			regSetup: func() *cs.ChangesetsRegistry {
				reg := cs.NewChangesetsRegistry()
				reg.Add("0001_test", cs.Configure(&templateStubChangeset{}).WithConfigResolver(inputTypeResolverForTest))

				return reg
			},
			rmSetup: func() *fresolvers.ConfigResolverManager {
				rm := fresolvers.NewConfigResolverManager()
				rm.Register(inputTypeResolverForTest, fresolvers.ResolverInfo{Description: "inputType"})

				return rm
			},
			checkGot: func(t *testing.T, got string) {
				t.Helper()
				require.Equal(t, "# Generated via template-input command\nenvironment: testnet\ndomain: mydomain\nchangesets:\n  # Config Resolver: github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/pipeline/template.inputTypeResolverForTest\n  # Input type: template.InputStruct\n  - 0001_test:\n      # Optional: Chain overrides (uncomment if needed)\n      # chainOverrides:\n      #   - 1  # Chain selector 1\n      #   - 2  # Chain selector 2\n      payload:\n        chain: # string\n        value: # int\n", got)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := GenerateMultiChangesetYAML(tt.domainName, tt.envKey, tt.changesetNames, tt.regSetup(), tt.rmSetup(), 5)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Equal(t, tt.wantErr, err.Error())

				return
			}
			require.NoError(t, err)
			if tt.checkGot != nil {
				tt.checkGot(t, got)
			}
		})
	}
}

func TestGenerateStructYAMLWithDepthLimit_Struct(t *testing.T) {
	t.Parallel()

	type S struct {
		A string `yaml:"a"`
		B int    `yaml:"b"`
	}

	got, err := GenerateStructYAMLWithDepthLimit(reflect.TypeOf(S{}), "  ", 0, make(map[reflect.Type]bool), 5)
	require.NoError(t, err)
	require.Equal(t, "  a: # string\n  b: # int\n", got)
}

func TestGenerateStructYAMLWithDepthLimit_DepthExceeded(t *testing.T) {
	t.Parallel()

	type S struct {
		A string `yaml:"a"`
	}

	got, err := GenerateStructYAMLWithDepthLimit(reflect.TypeOf(S{}), "  ", 10, make(map[reflect.Type]bool), 5)
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestGenerateStructYAMLWithDepthLimit_CircularRef(t *testing.T) {
	t.Parallel()

	type Node struct {
		Next *Node `yaml:"next"`
	}

	got, err := GenerateStructYAMLWithDepthLimit(reflect.TypeOf(Node{}), "  ", 0, make(map[reflect.Type]bool), 10)
	require.NoError(t, err)
	require.Equal(t, "  next:\n# ... (circular reference to template.Node)\n", got)
}

func TestGenerateFieldValueWithDepthLimit_String(t *testing.T) {
	t.Parallel()

	got, err := GenerateFieldValueWithDepthLimit(reflect.TypeOf(""), "  ", 0, make(map[reflect.Type]bool), 5)
	require.NoError(t, err)
	require.Equal(t, " # string", got)
}

func TestGenerateFieldValueWithDepthLimit_Int(t *testing.T) {
	t.Parallel()

	got, err := GenerateFieldValueWithDepthLimit(reflect.TypeOf(0), "  ", 0, make(map[reflect.Type]bool), 5)
	require.NoError(t, err)
	require.Equal(t, " # int", got)
}

func TestGenerateFieldValueWithDepthLimit_Slice(t *testing.T) {
	t.Parallel()

	got, err := GenerateFieldValueWithDepthLimit(reflect.TypeOf([]string{}), "  ", 0, make(map[reflect.Type]bool), 5)
	require.NoError(t, err)
	require.Equal(t, "\n  - # string", got)
}

func TestGetFieldName_YAMLTag(t *testing.T) {
	t.Parallel()

	f := reflect.StructField{
		Name: "MyField",
		Tag:  reflect.StructTag(`yaml:"my_field"`),
	}
	require.Equal(t, "my_field", GetFieldName(f))
}

func TestGetFieldName_JSONTag(t *testing.T) {
	t.Parallel()

	f := reflect.StructField{
		Name: "MyField",
		Tag:  reflect.StructTag(`json:"myField"`),
	}
	require.Equal(t, "myField", GetFieldName(f))
}

func TestGetFieldName_NoTag(t *testing.T) {
	t.Parallel()

	f := reflect.StructField{
		Name: "MyField",
		Tag:  "",
	}
	require.Equal(t, "myfield", GetFieldName(f))
}

func TestGetFieldName_IgnoreTag(t *testing.T) {
	t.Parallel()

	// When yaml is "-", GetFieldName returns "-"
	f := reflect.StructField{
		Name: "MyField",
		Tag:  reflect.StructTag(`yaml:"-"`),
	}
	require.Equal(t, "-", GetFieldName(f))

	// json:",omitempty" - parts[0] is "" so we fall through to ToLower(Name)
	f2 := reflect.StructField{
		Name: "OtherField",
		Tag:  reflect.StructTag(`json:",omitempty"`),
	}
	require.Equal(t, "otherfield", GetFieldName(f2))
}
