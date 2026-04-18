package service

import "testing"

func TestNormalizeWaMessageFragmentForEvolution_doubleMessageWrapper(t *testing.T) {
	inner := map[string]interface{}{
		"audioMessage": map[string]interface{}{
			"url": "https://example.invalid/x.ogg",
		},
	}
	double := map[string]interface{}{
		"message": map[string]interface{}{
			"message": inner,
		},
	}
	got := normalizeWaMessageFragmentForEvolution(double)
	if got == nil {
		t.Fatal("nil")
	}
	am, ok := got["audioMessage"].(map[string]interface{})
	if !ok || am == nil {
		t.Fatalf("expected audioMessage map, got %#v", got)
	}
	if am["url"] != "https://example.invalid/x.ogg" {
		t.Fatalf("url: %#v", am["url"])
	}
}

func TestNormalizeWaMessageFragmentForEvolution_alreadyFlat(t *testing.T) {
	m := map[string]interface{}{
		"audioMessage": map[string]interface{}{"seconds": float64(3)},
	}
	got := normalizeWaMessageFragmentForEvolution(m)
	am, ok := got["audioMessage"].(map[string]interface{})
	if !ok || am == nil || am["seconds"] != float64(3) {
		t.Fatalf("unexpected: %#v", got)
	}
}
