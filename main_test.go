package googlenewsdecoder

import "testing"

func TestGetBase64Str(t *testing.T) {
	decoder, err := NewGoogleDecoder("")
	if err != nil {
		t.Fatalf("NewGoogleDecoder() error = %v", err)
	}

	tests := []struct {
		name    string
		source  string
		want    string
		wantErr bool
	}{
		{
			name:   "articles format",
			source: "https://news.google.com/articles/CBMiTESTVALUE?hl=en-US&gl=US&ceid=US:en",
			want:   "CBMiTESTVALUE",
		},
		{
			name:   "rss read format",
			source: "https://news.google.com/rss/read/CBMiREADVALUE?oc=5",
			want:   "CBMiREADVALUE",
		},
		{
			name:    "invalid host",
			source:  "https://example.com/articles/CBMiTESTVALUE",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decoder.GetBase64Str(tc.source)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("GetBase64Str() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("GetBase64Str() unexpected error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("GetBase64Str() = %q, want %q", got, tc.want)
			}
		})
	}
}
