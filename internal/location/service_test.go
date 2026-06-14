package location

import "testing"

func TestNormalizeLocationMarksGPSPreciseAndConsented(t *testing.T) {
	lat := 23.8103
	lng := 90.4125
	source := "gps"

	result, err := normalizeUpdateRequest(UpdateLocationRequest{
		Latitude:  &lat,
		Longitude: &lng,
		Source:    &source,
	})
	if err != nil {
		t.Fatalf("expected valid GPS location: %v", err)
	}
	if result.Source != SourceGPS || !result.IsPrecise || result.LocationConsentAt == nil {
		t.Fatalf("unexpected GPS normalization: %#v", result)
	}
}

func TestNormalizeLocationMarksIPLowTrust(t *testing.T) {
	lat := 23.8103
	lng := 90.4125
	source := "IP"

	result, err := normalizeUpdateRequest(UpdateLocationRequest{
		Latitude:  &lat,
		Longitude: &lng,
		Source:    &source,
	})
	if err != nil {
		t.Fatalf("expected valid IP location: %v", err)
	}
	if result.IsPrecise || result.LocationConsentAt != nil {
		t.Fatalf("expected IP location to be low trust: %#v", result)
	}
}

func TestNormalizeLocationRejectsInvalidAccuracy(t *testing.T) {
	lat := 23.8103
	lng := 90.4125
	source := "GPS"
	accuracy := 10001.0

	_, err := normalizeUpdateRequest(UpdateLocationRequest{
		Latitude:       &lat,
		Longitude:      &lng,
		AccuracyMeters: &accuracy,
		Source:         &source,
	})
	if err == nil {
		t.Fatal("expected invalid accuracy to fail")
	}
}
