package p2pkey

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	peerIDStr           = "12D3KooWM1111111111111111111111111111111111111111111"
	peerIDWithPrefixStr = "p2p_12D3KooWM1111111111111111111111111111111111111111111"
)

func Test_MakePeerID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    string
		want    string
		wantErr bool
	}{
		{
			name: "valid peerID",
			give: peerIDStr,
			want: peerIDWithPrefixStr,
		},
		{
			name:    "invalid peerID",
			give:    "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, err := MakePeerID(tt.give)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, id.String())
			}
		})
	}
}

func Test_PeerID_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    string
		want    string
		wantErr bool
	}{
		{
			name: "valid peerID",
			give: peerIDStr,
			want: peerIDWithPrefixStr,
		},
		{
			name: "empty peerID",
			give: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, err := MakePeerID(tt.give)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, id.String())
			}
		})
	}
}

func Test_PeerID_Raw(t *testing.T) {
	t.Parallel()

	id, err := MakePeerID(peerIDStr)
	require.NoError(t, err)
	assert.Equal(t, peerIDStr, id.Raw())
}

func Test_PeerID_UnmarshalString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    string
		want    string
		wantErr bool
	}{
		{
			name: "valid peerID",
			give: peerIDStr,
			want: peerIDWithPrefixStr,
		},
		{
			name:    "invalid peerID",
			give:    "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id := PeerID{}
			err := id.UnmarshalString(tt.give)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, id.String())
			}
		})
	}
}

func Test_PeerID_MarshalText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		give string
		want []byte
	}{
		{
			name: "valid peerID with prefix",
			give: peerIDWithPrefixStr,
			want: []byte(peerIDStr),
		},
		{
			name: "valid peerID without prefix",
			give: peerIDStr,
			want: []byte(peerIDStr),
		},
		{
			name: "empty peerID",
			give: "",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, err := MakePeerID(tt.give)
			require.NoError(t, err)

			b, err := id.MarshalText()
			require.NoError(t, err)
			assert.Equal(t, tt.want, b)
		})
	}
}

func Test_PeerID_UnmarshalText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    string
		want    string
		wantErr bool
	}{
		{
			name: "valid peerID with prefix",
			give: peerIDWithPrefixStr,
			want: peerIDWithPrefixStr,
		},
		{
			name: "valid peerID without prefix",
			give: peerIDStr,
			want: peerIDWithPrefixStr,
		},
		{
			name: "empty peerID",
			give: "",
			want: "",
		},
		{
			name:    "invalid peerID",
			give:    "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := PeerID{}
			err := got.UnmarshalText([]byte(tt.give))

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got.String())
			}
		})
	}
}
