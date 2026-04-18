package service

import "testing"

func TestSniffAudioMIME(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"empty", nil, ""},
		{"short", []byte{1, 2, 3}, ""},
		{"ogg", []byte("OggS\x00\x00\x00\x00"), "audio/ogg"},
		{"webm_ebml", []byte{0x1a, 0x45, 0xdf, 0xa3, 0, 0, 0, 0}, "audio/webm"},
		{"mp4_ftyp", []byte{0, 0, 0, 0x20, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm'}, "audio/mp4"},
		{"wav", []byte("RIFF\x00\x00\x00\x00WAVE\x00\x00"), "audio/wav"},
		{"id3", []byte("ID3\x04"), "audio/mpeg"},
		{"mpeg_sync", []byte{0xff, 0xfb, 0, 0}, "audio/mpeg"},
		{"json_like", []byte(`{"error":`), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SniffAudioMIME(tt.data)
			if got != tt.want {
				t.Fatalf("SniffAudioMIME(%q) = %q, want %q", tt.data, got, tt.want)
			}
		})
	}
}
