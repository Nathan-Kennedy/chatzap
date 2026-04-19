package handler

import "testing"

func TestMapEvolutionStatus_disconnectedNotConnected(t *testing.T) {
	// Regressão: "disconnected" contém a substring "connected" — não pode mapear para connected.
	if g := mapEvolutionStatus("close", "disconnected"); g != "disconnected" {
		t.Fatalf("close+disconnected: got %q want disconnected", g)
	}
	if g := mapEvolutionStatus("disconnected", "disconnected"); g != "disconnected" {
		t.Fatalf("disconnected+disconnected: got %q want disconnected", g)
	}
	if g := mapEvolutionStatus("close", ""); g != "disconnected" {
		t.Fatalf("close+empty: got %q want disconnected", g)
	}
}

func TestMapEvolutionStatus_connected(t *testing.T) {
	if g := mapEvolutionStatus("open", "connected"); g != "connected" {
		t.Fatalf("open+connected: got %q want connected", g)
	}
	if g := mapEvolutionStatus("", "connected"); g != "connected" {
		t.Fatalf("empty+connected bool: got %q want connected", g)
	}
}

func TestMapEvolutionStatus_qrPending(t *testing.T) {
	if g := mapEvolutionStatus("qr", "disconnected"); g != "qr_pending" {
		t.Fatalf("qr: got %q want qr_pending", g)
	}
}
