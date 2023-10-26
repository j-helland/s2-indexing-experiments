package main

import (
	"math"
	"math/rand"

	"github.com/golang/geo/r1"
	"github.com/golang/geo/s1"
	"github.com/golang/geo/s2"
)

// Encapsulation of RNG state to maintain consistency across benchmarks.
type BenchRNG struct {
	rng rand.Rand
}

func NewBenchRNG(seed int64) *BenchRNG {
	return &BenchRNG{*rand.New(rand.NewSource(seed))}
}

// randomBits returns a 64-bit random unsigned integer whose lowest "num" are random, and
// whose other bits are zero.
func (r *BenchRNG) randomBits(num uint32) uint64 {
	// Make sure the request is for not more than 63 bits.
	if num > 63 {
		num = 63
	}
	return uint64(r.rng.Int63()) & ((1 << num) - 1)
}

// randomFloat64 returns a uniformly distributed value in the range [0,1).
// Note that the values returned are all multiples of 2**-53, which means that
// not all possible values in this range are returned.
func (r *BenchRNG) randomFloat64() float64 {
	const randomFloatBits = 53
	return math.Ldexp(float64(r.randomBits(randomFloatBits)), -randomFloatBits)
}

// randomUniformInt returns a uniformly distributed integer in the range [0,n).
// NOTE: This is replicated here to stay in sync with how the C++ code generates
// uniform randoms. (instead of using Go's math/rand package directly).
func (r *BenchRNG) randomUniformInt(n int) int {
	return int(r.randomFloat64() * float64(n))
}

// randomUniformFloat64 returns a uniformly distributed value in the range [min, max).
func (r *BenchRNG) randomUniformFloat64(min, max float64) float64 {
	return min + r.randomFloat64()*(max-min)
}

// randomPoint returns a random unit-length vector.
func (r *BenchRNG) randomPoint() s2.Point {
	return s2.PointFromCoords(r.randomUniformFloat64(-1, 1),
		r.randomUniformFloat64(-1, 1), r.randomUniformFloat64(-1, 1))
}

func (r *BenchRNG) randomRect() s2.Rect {
	ll1 := s2.LatLngFromPoint(r.randomPoint())
	ll2 := s2.LatLngFromPoint(r.randomPoint())
	lat_min, lat_max := min(ll1.Lat.Radians(), ll2.Lat.Radians()), max(ll1.Lat.Radians(), ll2.Lat.Radians())
	lng_min, lng_max := min(ll1.Lng.Radians(), ll2.Lng.Radians()), max(ll1.Lng.Radians(), ll2.Lng.Radians())
	return s2.Rect{
		Lat: r1.Interval{Lo: lat_min, Hi: lat_max},
		Lng: s1.Interval{Lo: lng_min, Hi: lng_max},
	}
}

// samplePointFromRect returns a point chosen uniformly at random (with respect
// to area on the sphere) from the given rectangle.
func (r *BenchRNG) samplePointFromRect(rect s2.Rect) s2.Point {
	// First choose a latitude uniformly with respect to area on the sphere.
	sinLo := math.Sin(rect.Lat.Lo)
	sinHi := math.Sin(rect.Lat.Hi)
	lat := math.Asin(r.randomUniformFloat64(sinLo, sinHi))

	// Now choose longitude uniformly within the given range.
	lng := rect.Lng.Lo + r.randomFloat64()*rect.Lng.Length()

	return s2.PointFromLatLng(s2.LatLng{
		Lat: s1.Angle(lat),
		Lng: s1.Angle(lng),
	}.Normalized())
}
