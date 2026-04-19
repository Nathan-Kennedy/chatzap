package service

import "testing"

func TestNormalizeToE164(t *testing.T) {
	tests := []struct {
		in    string
		want  string
		wantErr bool
	}{
		{"+351912345678", "+351912345678", false},
		{"351912345678", "+351912345678", false},
		{" +55 (11) 91234-5678 ", "+5511912345678", false},
		{"", "", true},
		{"+++", "", true},
		{"+12", "", true},
	}
	for _, tc := range tests {
		got, err := NormalizeToE164(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("NormalizeToE164(%q) err=nil want erro", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("NormalizeToE164(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("NormalizeToE164(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}
