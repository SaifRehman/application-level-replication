package identity

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIdentityValidate(t *testing.T) {
	tests := []struct {
		name    string
		in      Identity
		wantErr string
	}{
		{
			name: "valid full identity",
			in: Identity{
				FullName:   "Ada Lovelace",
				Phone:      "+1-202-555-0143",
				Address:    "1 Mill St",
				Email:      "ada@example.com",
				PassportNo: "P1234567",
			},
		},
		{
			name: "valid with only full name",
			in:   Identity{FullName: "Ada"},
		},
		{
			name:    "missing full name",
			in:      Identity{Email: "ada@example.com"},
			wantErr: "full_name is required",
		},
		{
			name:    "blank full name",
			in:      Identity{FullName: "   "},
			wantErr: "full_name is required",
		},
		{
			name:    "invalid email",
			in:      Identity{FullName: "Ada", Email: "not-an-email"},
			wantErr: "email is invalid",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.in.Validate()
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErr)
		})
	}
}
