package service

// SniffAudioMIME infere audio/* a partir dos primeiros bytes (WhatsApp pode enviar OGG/Opus, MP4/AAC, WebM, etc.).
// Quando a Evolution devolve mimetype vazio ou genérico, isto evita forçar OGG no browser e na API Gemini.
func SniffAudioMIME(b []byte) string {
	if len(b) < 4 {
		return ""
	}
	if string(b[0:4]) == "OggS" {
		return "audio/ogg"
	}
	if b[0] == 0x1a && b[1] == 0x45 && b[2] == 0xdf && b[3] == 0xa3 {
		return "audio/webm"
	}
	if len(b) >= 12 && string(b[4:8]) == "ftyp" {
		return "audio/mp4"
	}
	if len(b) >= 12 && string(b[0:4]) == "RIFF" && string(b[8:12]) == "WAVE" {
		return "audio/wav"
	}
	if len(b) >= 3 && string(b[0:3]) == "ID3" {
		return "audio/mpeg"
	}
	if len(b) >= 2 && b[0] == 0xff && (b[1]&0xe0) == 0xe0 {
		return "audio/mpeg"
	}
	return ""
}
