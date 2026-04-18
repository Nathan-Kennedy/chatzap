package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func TestWriteMessageMediaBytes_oggFromMime(t *testing.T) {
	dir := t.TempDir()
	id := uuid.MustParse("00000000-0000-0000-0000-000000000042")
	rel, err := WriteMessageMediaBytes(dir, id, []byte("fake"), "", "audio/ogg; codecs=opus")
	if err != nil {
		t.Fatal(err)
	}
	if rel != id.String()+".ogg" {
		t.Fatalf("rel=%q", rel)
	}
	p := filepath.Join(dir, rel)
	b, err := os.ReadFile(p)
	if err != nil || string(b) != "fake" {
		t.Fatalf("read: %v %q", err, b)
	}
}
