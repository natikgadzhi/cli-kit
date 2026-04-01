package auth

import "testing"

func TestMaskToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  string
	}{
		{
			name:  "empty token",
			token: "",
			want:  "(none)",
		},
		{
			name:  "very short token (3 chars)",
			token: "abc",
			want:  "••••",
		},
		{
			name:  "short token (4 chars)",
			token: "abcd",
			want:  "••••",
		},
		{
			name:  "medium token (8 chars)",
			token: "abcdefgh",
			want:  "ab•••gh",
		},
		{
			name:  "boundary token (11 chars)",
			token: "abcdefghijk",
			want:  "ab•••jk",
		},
		{
			name:  "boundary token (12 chars)",
			token: "abcdefghijkl",
			want:  "abcd•••ijkl",
		},
		{
			name:  "normal token (20 chars)",
			token: "ghp_1234567890abcdef",
			want:  "ghp_•••cdef",
		},
		{
			name:  "very long token (40 chars)",
			token: "fmu1-abcdefghijklmnopqrstuvwxyz1234567890",
			want:  "fmu1•••7890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskToken(tt.token)
			if got != tt.want {
				t.Errorf("MaskToken(%q) = %q, want %q", tt.token, got, tt.want)
			}
		})
	}
}
