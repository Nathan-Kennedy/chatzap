package handler

import "testing"

func TestAttachmentResponseContentType_audioNoHintUsesOctetStream(t *testing.T) {
	got := attachmentResponseContentType("audio", "", "")
	if got != "application/octet-stream" {
		t.Fatalf("got %q want application/octet-stream", got)
	}
}

func TestAttachmentResponseContentType_audioSniffOgg(t *testing.T) {
	got := attachmentResponseContentType("audio", "audio/ogg; codecs=opus", "audio/ogg")
	if got != "audio/ogg" {
		t.Fatalf("got %q", got)
	}
}

func TestAttachmentResponseContentType_imageDefault(t *testing.T) {
	got := attachmentResponseContentType("image", "", "")
	if got != "image/jpeg" {
		t.Fatalf("got %q", got)
	}
}
