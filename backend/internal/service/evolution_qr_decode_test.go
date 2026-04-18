package service

import "testing"

func TestDecodeQR_prefersBase64OverBaileysRef(t *testing.T) {
	raw := []byte(`{
		"code": "2@xyznotanimage",
		"base64": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
	}`)
	out, err := decodeQR(raw)
	if err != nil {
		t.Fatal(err)
	}
	if out.Code == "" || out.Code == "2@xyznotanimage" {
		t.Fatalf("expected base64 image payload, got %q", out.Code)
	}
}

func TestDecodeQR_pairingOnly(t *testing.T) {
	raw := []byte(`{"pairingCode":"ABCD-EFGH"}`)
	out, err := decodeQR(raw)
	if err != nil {
		t.Fatal(err)
	}
	if out.PairingCode != "ABCD-EFGH" || out.Code != "" {
		t.Fatalf("got %+v", out)
	}
}
