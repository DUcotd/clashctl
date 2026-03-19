package system

import "testing"

func TestParseEgressInfo(t *testing.T) {
	info, err := parseEgressInfo([]byte(`{"ip":"203.0.113.8","country":"United States","country_iso":"US"}`))
	if err != nil {
		t.Fatalf("parseEgressInfo() error: %v", err)
	}
	if info.IP != "203.0.113.8" || info.Country != "United States" || info.CountryCode != "US" {
		t.Fatalf("parseEgressInfo() = %#v", info)
	}
}

func TestParseEgressInfoFallbackCountryCode(t *testing.T) {
	info, err := parseEgressInfo([]byte(`{"ip":"203.0.113.9","country":"Singapore","country_code":"sg"}`))
	if err != nil {
		t.Fatalf("parseEgressInfo() error: %v", err)
	}
	if info.CountryCode != "SG" {
		t.Fatalf("CountryCode = %q, want SG", info.CountryCode)
	}
}

func TestParseEgressInfoRejectsEmptyPayload(t *testing.T) {
	if _, err := parseEgressInfo([]byte(`{"hello":"world"}`)); err == nil {
		t.Fatal("expected parseEgressInfo() to reject empty payload")
	}
}
