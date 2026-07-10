package evm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkflowIdsExpression(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		fi      *FunctionInfo
		want    string
		wantErr string
	}{
		{
			name: "single workflow ID as only parameter",
			fi: &FunctionInfo{
				Parameters: []ParameterInfo{{Name: "workflowId", GoType: "[32]byte"}},
			},
			want: "[][32]byte{args}",
		},
		{
			name: "single workflow ID in args struct",
			fi: &FunctionInfo{
				Parameters: []ParameterInfo{
					{Name: "workflowName", GoType: "string"},
					{Name: "workflowId", GoType: "[32]byte"},
				},
			},
			want: "[][32]byte{args.WorkflowId}",
		},
		{
			name: "multiple workflow IDs as only parameter",
			fi: &FunctionInfo{
				Parameters: []ParameterInfo{{Name: "workflowIds", GoType: "[][32]byte"}},
			},
			want: "args",
		},
		{
			name: "multiple workflow IDs in args struct",
			fi: &FunctionInfo{
				Parameters: []ParameterInfo{
					{Name: "workflowIds", GoType: "[][32]byte"},
					{Name: "donFamily", GoType: "string"},
				},
			},
			want: "args.WorkflowIds",
		},
		{
			name: "missing workflow ID",
			fi: &FunctionInfo{
				Parameters: []ParameterInfo{{Name: "id", GoType: "[32]byte"}},
			},
			wantErr: `missing "workflowId" or "workflowIds" parameter`,
		},
		{
			name: "wrong workflow IDs type",
			fi: &FunctionInfo{
				Parameters: []ParameterInfo{{Name: "workflowIds", GoType: "[32]byte"}},
			},
			wantErr: `parameter "workflowIds" must be [][32]byte, got [32]byte`,
		},
		{
			name: "wrong workflow ID type",
			fi: &FunctionInfo{
				Parameters: []ParameterInfo{{Name: "workflowId", GoType: "[][32]byte"}},
			},
			wantErr: `parameter "workflowId" must be [32]byte, got [][32]byte`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := workflowIdsExpression(test.fi)
			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				require.Empty(t, got)

				return
			}

			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}
