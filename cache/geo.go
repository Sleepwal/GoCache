package cache

import (
	"fmt"
	"math"
	"sort"
	"time"
)

const (
	earthRadiusKm = 6371.0
	geoHashBits   = 52
	geoStepBits   = 26
)

type GeoUnit string

const (
	UnitM     GeoUnit = "m"
	UnitKm    GeoUnit = "km"
	UnitMi    GeoUnit = "mi"
	UnitFt    GeoUnit = "ft"
)

type GeoMember struct {
	Name      string
	Longitude float64
	Latitude  float64
}

type GeoSearchResult struct {
	Name      string
	Distance  float64
	Longitude float64
	Latitude  float64
}

type geoData struct {
	members map[string]*geoMemberEntry
}

type geoMemberEntry struct {
	longitude float64
	latitude  float64
	geohash   int64
}

func newGeoData() *geoData {
	return &geoData{
		members: make(map[string]*geoMemberEntry),
	}
}

func encodeGeoHash(lng, lat float64) int64 {
	latRange := [2]float64{-90, 90}
	lngRange := [2]float64{-180, 180}

	var hash int64
	for i := geoHashBits - 1; i >= 0; i-- {
		if (geoHashBits-1-i)%2 == 0 {
			mid := (lngRange[0] + lngRange[1]) / 2
			if lng >= mid {
				hash |= 1 << uint(i)
				lngRange[0] = mid
			} else {
				lngRange[1] = mid
			}
		} else {
			mid := (latRange[0] + latRange[1]) / 2
			if lat >= mid {
				hash |= 1 << uint(i)
				latRange[0] = mid
			} else {
				latRange[1] = mid
			}
		}
	}
	return hash
}

func decodeGeoHash(hash int64) (lng, lat float64) {
	latRange := [2]float64{-90, 90}
	lngRange := [2]float64{-180, 180}

	for i := geoHashBits - 1; i >= 0; i-- {
		bit := (hash >> uint(i)) & 1
		if (geoHashBits-1-i)%2 == 0 {
			mid := (lngRange[0] + lngRange[1]) / 2
			if bit == 1 {
				lngRange[0] = mid
			} else {
				lngRange[1] = mid
			}
		} else {
			mid := (latRange[0] + latRange[1]) / 2
			if bit == 1 {
				latRange[0] = mid
			} else {
				latRange[1] = mid
			}
		}
	}

	lng = (lngRange[0] + lngRange[1]) / 2
	lat = (latRange[0] + latRange[1]) / 2
	return
}

func haversineDistance(lng1, lat1, lng2, lat2 float64) float64 {
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

func convertDistance(km float64, unit GeoUnit) float64 {
	switch unit {
	case UnitM:
		return km * 1000
	case UnitKm:
		return km
	case UnitMi:
		return km / 1.60934
	case UnitFt:
		return km * 3280.84
	default:
		return km
	}
}

type GeoCache struct {
	cache         *MemoryCache
	sortedSetCache *SortedSetCache
}

func NewGeoCache(cache *MemoryCache) *GeoCache {
	if cache == nil {
		cache = New()
	}
	return &GeoCache{
		cache:          cache,
		sortedSetCache: NewSortedSetCacheWithMemory(cache),
	}
}

func (gc *GeoCache) getOrCreateGeoData(key string) (*geoData, bool) {
	val, found := gc.cache.Get(key)
	if !found {
		return nil, false
	}
	gd, ok := val.(*geoData)
	if !ok {
		return nil, false
	}
	return gd, true
}

func (gc *GeoCache) GeoAdd(key string, members ...GeoMember) (int, error) {
	gc.cache.mu.Lock()
	defer gc.cache.mu.Unlock()

	var gd *geoData
	item, exists := gc.cache.items[key]
	if exists && !item.IsExpired() {
		var ok bool
		gd, ok = item.Value.(*geoData)
		if !ok {
			gd = newGeoData()
		}
	} else {
		gd = newGeoData()
	}

	added := 0
	for _, m := range members {
		if m.Longitude < -180 || m.Longitude > 180 || m.Latitude < -85.05112878 || m.Latitude > 85.05112878 {
			continue
		}

		geohash := encodeGeoHash(m.Longitude, m.Latitude)
		if _, exists := gd.members[m.Name]; !exists {
			added++
		}
		gd.members[m.Name] = &geoMemberEntry{
			longitude: m.Longitude,
			latitude:  m.Latitude,
			geohash:   geohash,
		}
	}

	gc.cache.items[key] = &Item{
		Value:      gd,
		Expiration: 0,
		LastAccess: time.Now().UnixNano(),
	}

	return added, nil
}

func (gc *GeoCache) GeoDist(key, member1, member2 string, unit GeoUnit) (float64, error) {
	gc.cache.mu.RLock()
	defer gc.cache.mu.RUnlock()

	gd, found := gc.getOrCreateGeoDataRLocked(key)
	if !found {
		return 0, nil
	}

	m1, ok1 := gd.members[member1]
	m2, ok2 := gd.members[member2]
	if !ok1 || !ok2 {
		return 0, nil
	}

	dist := haversineDistance(m1.longitude, m1.latitude, m2.longitude, m2.latitude)
	return convertDistance(dist, unit), nil
}

func (gc *GeoCache) getOrCreateGeoDataRLocked(key string) (*geoData, bool) {
	item, exists := gc.cache.items[key]
	if !exists || item.IsExpired() {
		return nil, false
	}
	gd, ok := item.Value.(*geoData)
	if !ok {
		return nil, false
	}
	return gd, true
}

func (gc *GeoCache) GeoHash(key string, members ...string) ([]string, error) {
	gc.cache.mu.RLock()
	defer gc.cache.mu.RUnlock()

	gd, found := gc.getOrCreateGeoDataRLocked(key)
	if !found {
		result := make([]string, len(members))
		return result, nil
	}

	result := make([]string, len(members))
	for i, m := range members {
		entry, ok := gd.members[m]
		if !ok {
			result[i] = ""
			continue
		}
		result[i] = fmt.Sprintf("%013x", entry.geohash)
	}
	return result, nil
}

func (gc *GeoCache) GeoPos(key string, members ...string) ([]*GeoMember, error) {
	gc.cache.mu.RLock()
	defer gc.cache.mu.RUnlock()

	gd, found := gc.getOrCreateGeoDataRLocked(key)

	result := make([]*GeoMember, len(members))
	for i, m := range members {
		if !found {
			result[i] = nil
			continue
		}
		entry, ok := gd.members[m]
		if !ok {
			result[i] = nil
			continue
		}
		result[i] = &GeoMember{
			Name:      m,
			Longitude: entry.longitude,
			Latitude:  entry.latitude,
		}
	}
	return result, nil
}

func (gc *GeoCache) GeoRadius(key string, lng, lat, radius float64, unit GeoUnit, withCoord, withDist bool, count int) ([]GeoSearchResult, error) {
	gc.cache.mu.RLock()
	defer gc.cache.mu.RUnlock()

	gd, found := gc.getOrCreateGeoDataRLocked(key)
	if !found {
		return nil, nil
	}

	radiusKm := radius
	switch unit {
	case UnitM:
		radiusKm = radius / 1000
	case UnitMi:
		radiusKm = radius * 1.60934
	case UnitFt:
		radiusKm = radius / 3280.84
	}

	var results []GeoSearchResult
	for name, entry := range gd.members {
		dist := haversineDistance(lng, lat, entry.longitude, entry.latitude)
		if dist <= radiusKm {
			r := GeoSearchResult{
				Name:      name,
				Longitude: entry.longitude,
				Latitude:  entry.latitude,
			}
			if withDist {
				r.Distance = convertDistance(dist, unit)
			}
			results = append(results, r)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Distance < results[j].Distance
	})

	if count > 0 && len(results) > count {
		results = results[:count]
	}

	return results, nil
}

func (gc *GeoCache) GeoRadiusByMember(key, member string, radius float64, unit GeoUnit, withCoord, withDist bool, count int) ([]GeoSearchResult, error) {
	gc.cache.mu.RLock()
	gd, found := gc.getOrCreateGeoDataRLocked(key)
	if !found {
		gc.cache.mu.RUnlock()
		return nil, nil
	}

	entry, ok := gd.members[member]
	if !ok {
		gc.cache.mu.RUnlock()
		return nil, nil
	}
	lng := entry.longitude
	lat := entry.latitude
	gc.cache.mu.RUnlock()

	return gc.GeoRadius(key, lng, lat, radius, unit, withCoord, withDist, count)
}
