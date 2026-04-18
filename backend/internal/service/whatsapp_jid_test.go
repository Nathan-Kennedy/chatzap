package service

import "testing"

func TestNormalizeContactJID_plainBRMobile(t *testing.T) {
	got := NormalizeContactJID("69993378283")
	want := "5569993378283@s.whatsapp.net"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestNormalizeContactJID_e164(t *testing.T) {
	got := NormalizeContactJID("5569993378283")
	want := "5569993378283@s.whatsapp.net"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestNormalizeContactJID_existingJID(t *testing.T) {
	got := NormalizeContactJID("5569993378283@s.whatsapp.net")
	want := "5569993378283@s.whatsapp.net"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

// Webhook Evolution por vezes manda 12 dígitos (55+DDD+8) em falta do 9 do móvel.
func TestNormalizeContactJID_brMobileMissingNine(t *testing.T) {
	got := NormalizeContactJID("556993378283@s.whatsapp.net")
	want := "5569993378283@s.whatsapp.net"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestCollectJIDLookupKeys_br12and13Variants(t *testing.T) {
	keys := CollectJIDLookupKeys("556993378283@s.whatsapp.net", "")
	found12 := false
	found13 := false
	for _, k := range keys {
		if k == "556993378283@s.whatsapp.net" {
			found12 = true
		}
		if k == "5569993378283@s.whatsapp.net" {
			found13 = true
		}
	}
	if !found12 || !found13 {
		t.Fatalf("expected 12 and 13 digit JIDs, got %v", keys)
	}
}

func TestNormalizeContactJID_deviceSuffix(t *testing.T) {
	got := NormalizeContactJID("5569993378283:72@s.whatsapp.net")
	want := "5569993378283@s.whatsapp.net"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestNormalizeContactJID_group(t *testing.T) {
	got := NormalizeContactJID("120363123@g.us")
	if got != "120363123@g.us" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeContactJID_lid(t *testing.T) {
	got := NormalizeContactJID("123456789012345@lid")
	if got != "123456789012345@lid" {
		t.Fatalf("got %q want digits@lid", got)
	}
}

func TestCollectJIDLookupKeys_prefersAltPN(t *testing.T) {
	keys := CollectJIDLookupKeys("999@lid", "5569993378283@s.whatsapp.net")
	if len(keys) < 2 {
		t.Fatalf("expected 2 keys, got %v", keys)
	}
	if keys[0] != "5569993378283@s.whatsapp.net" {
		t.Fatalf("first key should be PN, got %q", keys[0])
	}
}

func TestInboundCanonicalJID_usesAlt(t *testing.T) {
	got := InboundCanonicalJID("111@lid", "5569993378283@s.whatsapp.net")
	if got != "5569993378283@s.whatsapp.net" {
		t.Fatalf("got %q", got)
	}
}
