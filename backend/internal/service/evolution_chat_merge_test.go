package service

import "testing"

func TestBodiesLikelySameMedia(t *testing.T) {
	if !bodiesLikelySameMedia("[imagem]", "[imagem]") {
		t.Fatal("placeholders")
	}
	if !bodiesLikelySameMedia("MASSA", "[imagem]") {
		t.Fatal("caption vs placeholder")
	}
	if !bodiesLikelySameMedia("[imagem]", "MASSA") {
		t.Fatal("placeholder vs caption")
	}
	if !bodiesLikelySameMedia("oi", "oi") {
		t.Fatal("exact")
	}
	if bodiesLikelySameMedia("a", "b") {
		t.Fatal("different captions should not match")
	}
}
