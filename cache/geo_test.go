package cache

import (
	"math"
	"testing"
)

func TestGeo_GeoAdd(t *testing.T) {
	c := New()
	gc := NewGeoCache(c)

	added, err := gc.GeoAdd("cities", GeoMember{Name: "beijing", Longitude: 116.40, Latitude: 39.90})
	if err != nil {
		t.Fatalf("GeoAdd failed: %v", err)
	}
	if added != 1 {
		t.Errorf("expected 1 added, got %d", added)
	}
}

func TestGeo_GeoAddMultiple(t *testing.T) {
	c := New()
	gc := NewGeoCache(c)

	added, err := gc.GeoAdd("cities",
		GeoMember{Name: "beijing", Longitude: 116.40, Latitude: 39.90},
		GeoMember{Name: "shanghai", Longitude: 121.47, Latitude: 31.23},
	)
	if err != nil {
		t.Fatalf("GeoAdd failed: %v", err)
	}
	if added != 2 {
		t.Errorf("expected 2 added, got %d", added)
	}
}

func TestGeo_GeoAddUpdate(t *testing.T) {
	c := New()
	gc := NewGeoCache(c)

	gc.GeoAdd("cities", GeoMember{Name: "beijing", Longitude: 116.40, Latitude: 39.90})
	added, _ := gc.GeoAdd("cities", GeoMember{Name: "beijing", Longitude: 117.00, Latitude: 40.00})

	if added != 0 {
		t.Errorf("expected 0 added for update, got %d", added)
	}
}

func TestGeo_GeoDist(t *testing.T) {
	c := New()
	gc := NewGeoCache(c)

	gc.GeoAdd("cities",
		GeoMember{Name: "beijing", Longitude: 116.40, Latitude: 39.90},
		GeoMember{Name: "shanghai", Longitude: 121.47, Latitude: 31.23},
	)

	dist, err := gc.GeoDist("cities", "beijing", "shanghai", UnitKm)
	if err != nil {
		t.Fatalf("GeoDist failed: %v", err)
	}

	expectedDist := 1068.0
	if math.Abs(dist-expectedDist)/expectedDist > 0.02 {
		t.Errorf("expected ~%.0f km, got %.2f km", expectedDist, dist)
	}
}

func TestGeo_GeoDistMeters(t *testing.T) {
	c := New()
	gc := NewGeoCache(c)

	gc.GeoAdd("cities",
		GeoMember{Name: "beijing", Longitude: 116.40, Latitude: 39.90},
		GeoMember{Name: "shanghai", Longitude: 121.47, Latitude: 31.23},
	)

	dist, _ := gc.GeoDist("cities", "beijing", "shanghai", UnitM)
	if dist < 1000000 {
		t.Errorf("expected > 1,000,000 m, got %.2f m", dist)
	}
}

func TestGeo_GeoDistNonExistent(t *testing.T) {
	c := New()
	gc := NewGeoCache(c)

	dist, _ := gc.GeoDist("nonexistent", "a", "b", UnitKm)
	if dist != 0 {
		t.Errorf("expected 0 for non-existent key, got %f", dist)
	}
}

func TestGeo_GeoPos(t *testing.T) {
	c := New()
	gc := NewGeoCache(c)

	gc.GeoAdd("cities", GeoMember{Name: "beijing", Longitude: 116.40, Latitude: 39.90})

	positions, err := gc.GeoPos("cities", "beijing")
	if err != nil {
		t.Fatalf("GeoPos failed: %v", err)
	}
	if len(positions) != 1 {
		t.Fatalf("expected 1 position, got %d", len(positions))
	}
	if positions[0] == nil {
		t.Fatal("expected non-nil position")
	}

	if math.Abs(positions[0].Longitude-116.40) > 0.01 {
		t.Errorf("longitude too far: got %f", positions[0].Longitude)
	}
	if math.Abs(positions[0].Latitude-39.90) > 0.01 {
		t.Errorf("latitude too far: got %f", positions[0].Latitude)
	}
}

func TestGeo_GeoPosNonExistent(t *testing.T) {
	c := New()
	gc := NewGeoCache(c)

	positions, _ := gc.GeoPos("nonexistent", "beijing")
	if positions[0] != nil {
		t.Error("expected nil for non-existent member")
	}
}

func TestGeo_GeoHash(t *testing.T) {
	c := New()
	gc := NewGeoCache(c)

	gc.GeoAdd("cities", GeoMember{Name: "beijing", Longitude: 116.40, Latitude: 39.90})

	hashes, err := gc.GeoHash("cities", "beijing")
	if err != nil {
		t.Fatalf("GeoHash failed: %v", err)
	}
	if len(hashes) != 1 {
		t.Fatalf("expected 1 hash, got %d", len(hashes))
	}
	if hashes[0] == "" {
		t.Error("expected non-empty geohash")
	}
}

func TestGeo_GeoRadius(t *testing.T) {
	c := New()
	gc := NewGeoCache(c)

	gc.GeoAdd("cities",
		GeoMember{Name: "beijing", Longitude: 116.40, Latitude: 39.90},
		GeoMember{Name: "tianjin", Longitude: 117.20, Latitude: 39.13},
		GeoMember{Name: "shanghai", Longitude: 121.47, Latitude: 31.23},
	)

	results, err := gc.GeoRadius("cities", 116.40, 39.90, 200, UnitKm, false, true, 0)
	if err != nil {
		t.Fatalf("GeoRadius failed: %v", err)
	}

	if len(results) < 2 {
		t.Errorf("expected at least 2 results within 200km of Beijing, got %d", len(results))
	}

	foundBeijing := false
	foundTianjin := false
	for _, r := range results {
		if r.Name == "beijing" {
			foundBeijing = true
		}
		if r.Name == "tianjin" {
			foundTianjin = true
		}
	}
	if !foundBeijing {
		t.Error("expected to find beijing within 200km")
	}
	if !foundTianjin {
		t.Error("expected to find tianjin within 200km")
	}
}

func TestGeo_GeoRadiusByMember(t *testing.T) {
	c := New()
	gc := NewGeoCache(c)

	gc.GeoAdd("cities",
		GeoMember{Name: "beijing", Longitude: 116.40, Latitude: 39.90},
		GeoMember{Name: "tianjin", Longitude: 117.20, Latitude: 39.13},
		GeoMember{Name: "shanghai", Longitude: 121.47, Latitude: 31.23},
	)

	results, err := gc.GeoRadiusByMember("cities", "beijing", 200, UnitKm, false, true, 0)
	if err != nil {
		t.Fatalf("GeoRadiusByMember failed: %v", err)
	}

	if len(results) < 2 {
		t.Errorf("expected at least 2 results, got %d", len(results))
	}
}

func TestGeo_GeoRadiusWithCount(t *testing.T) {
	c := New()
	gc := NewGeoCache(c)

	gc.GeoAdd("cities",
		GeoMember{Name: "beijing", Longitude: 116.40, Latitude: 39.90},
		GeoMember{Name: "tianjin", Longitude: 117.20, Latitude: 39.13},
		GeoMember{Name: "shanghai", Longitude: 121.47, Latitude: 31.23},
	)

	results, _ := gc.GeoRadius("cities", 116.40, 39.90, 2000, UnitKm, false, true, 2)
	if len(results) > 2 {
		t.Errorf("expected at most 2 results with count=2, got %d", len(results))
	}
}

func TestGeo_InvalidCoordinates(t *testing.T) {
	c := New()
	gc := NewGeoCache(c)

	added, _ := gc.GeoAdd("cities", GeoMember{Name: "invalid", Longitude: 200, Latitude: 100})
	if added != 0 {
		t.Error("expected 0 added for invalid coordinates")
	}
}

func TestGeo_EncodeDecodeHash(t *testing.T) {
	lng, lat := 116.40, 39.90
	hash := encodeGeoHash(lng, lat)
	decLng, decLat := decodeGeoHash(hash)

	if math.Abs(decLng-lng) > 0.01 {
		t.Errorf("longitude roundtrip error: got %f, expected %f", decLng, lng)
	}
	if math.Abs(decLat-lat) > 0.01 {
		t.Errorf("latitude roundtrip error: got %f, expected %f", decLat, lat)
	}
}
