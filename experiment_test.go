package main

import (
	"encoding/binary"
	"testing"

	"github.com/Workiva/go-datastructures/trie/xfast"
	"github.com/Workiva/go-datastructures/trie/yfast"
	"github.com/golang/geo/s2"
	"github.com/hashicorp/go-immutable-radix"
)

//====================================================================================================
// Helper junk.
//====================================================================================================
func cellToBytes(id s2.CellID) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(id))
	return b
}

// x/y-fast trie overhead
type Entry struct {
	key    uint64
	values []s2.LatLng
}

// x/y-fast trie overhead 
func (e *Entry) Key() uint64 {
	return e.key;
}

//====================================================================================================
// Benchmark setup.
//====================================================================================================
// Benchmarking constants.
const B_S2_LEVEL = 24
const B_NUM_REGIONS = 100				 // Number of disjoint spatial regions to sample points from.
const B_NUM_INSERTS = 1_000_000  // Number of random insertions to perform.

func (r *BenchRNG) sampleCoordinates(region s2.Rect, num int) []s2.LatLng {
	var coords []s2.LatLng
	for i := 0; i < num; i += 1 {
		coords = append(coords, s2.LatLngFromPoint(r.samplePointFromRect(region)))
	}
	return coords
}

func (r *BenchRNG) indexCoordinates(coords []s2.LatLng) []s2.CellID {
	// Uncomment to test with random S2 cells.
	//level := r.rng.Intn(31)
	level := B_S2_LEVEL
	var cells []s2.CellID
	for _, coord := range coords {
		cells = append(cells, s2.CellIDFromLatLng(coord).Parent(level))
	}
	return cells
}

func initBench(r *BenchRNG) ([]s2.LatLng, []s2.CellID) {
	const numPerRegion = B_NUM_INSERTS / B_NUM_REGIONS

	coords := make([]s2.LatLng, 0)
	indices := make([]s2.CellID, 0)
	for i := 0; i < B_NUM_REGIONS; i += 1 {
		region := r.randomRect()
		_coords := r.sampleCoordinates(region, numPerRegion)
		_indices := r.indexCoordinates(_coords)

		coords = append(coords, _coords...)
		indices = append(indices, _indices...)
	}
	return coords, indices
}

//====================================================================================================
// Insert benchmark.
//====================================================================================================
// std hashmap
func Benchmark_Hashmap_Insert(b *testing.B) {
	// setup
	coords, indices := initBench(NewBenchRNG(0))
	b.ResetTimer()

	// Benchmark.
	for i := 0; i < b.N; i += 1 {
		hashmapInsert(coords, indices)
	}
}

func hashmapInsert(coords []s2.LatLng, indices []s2.CellID) map[uint64][]s2.LatLng {
	table := make(map[uint64][]s2.LatLng)
	for i, coord := range coords {
		index := uint64(indices[i])
		if _, found := table[index]; !found {
			table[index] = make([]s2.LatLng, 0)
		}
		table[index] = append(table[index], coord)
	}
	return table
}

// radix tree
func Benchmark_Radix_Insert(b *testing.B) {
	// setup
	coords, indices := initBench(NewBenchRNG(0))
	b.ResetTimer()

	// Benchmark.
	for i := 0; i < b.N; i += 1 {
		radixInsert(coords, indices)
	}
}

func radixInsert(coords []s2.LatLng, indices []s2.CellID) *iradix.Tree {
	rt := iradix.New()
	for i, coord := range coords {
		index := cellToBytes(indices[i])
		if group, found := rt.Get(index); found {
			rt.Insert(index, append(group.([]s2.LatLng), coord))
		} else {
			rt.Insert(index, []s2.LatLng{coord})
		}
	}
	return rt
}

// x-fast trie
func Benchmark_XFastTrie_Insert(b *testing.B) {
	// setup
	coords, indices := initBench(NewBenchRNG(0))
	b.ResetTimer()

	// Benchmark.
	for i := 0; i < b.N; i += 1 {
		xfastInsert(coords, indices)
	}
}

func xfastInsert(coords []s2.LatLng, indices []s2.CellID) *xfast.XFastTrie {
	xft := xfast.New(uint64(0))
	for i, coord := range coords {
		index := uint64(indices[i])
		if e := xft.Get(index); e != nil {
			e.(*Entry).values = append(e.(*Entry).values, coord)
			xft.Insert(e)
		} else {
			xft.Insert(&Entry{index, []s2.LatLng{coord}})
		}
	}
	return xft
}

// y-fast trie
func Benchmark_YFastTrie_Insert(b *testing.B) {
	// setup
	coords, indices := initBench(NewBenchRNG(0))
	b.ResetTimer()

	// Benchmark.
	for i := 0; i < b.N; i += 1 {
		yfastInsert(coords, indices)
	}
}

func yfastInsert(coords []s2.LatLng, indices []s2.CellID) *yfast.YFastTrie {
	yft := yfast.New(uint64(0))
	for i, coord := range coords {
		index := uint64(indices[i])
		if e := yft.Get(index); e != nil {
			e.(*Entry).values = append(e.(*Entry).values, coord)
			yft.Insert(e)
		} else {
			yft.Insert(&Entry{index, []s2.LatLng{coord}})
		}
	}
	return yft
}

//====================================================================================================
// Lookup benchmark.
//====================================================================================================
// std hashmap
func Benchmark_Hashmap_Lookup(b *testing.B) {
	// setup
	r := NewBenchRNG(0)
	coords, indices := initBench(r)
	num_coords := len(coords)
	table := hashmapInsert(coords, indices)
	b.ResetTimer()

	// Benchmark
	for i := 0; i < b.N; i += 1 {
		index := indices[r.rng.Intn(num_coords)]
		_ = table[uint64(index)]
	}
}

// radix tree
func Benchmark_Radix_Lookup(b *testing.B) {
	// setup
	r := NewBenchRNG(0)
	coords, cells := initBench(r)
	num_coords := len(coords)
	rt := radixInsert(coords, cells)

	indices := make([][]byte, 0, num_coords)
	for _, cell := range cells {
		indices = append(indices, cellToBytes(cell))
	}
	b.ResetTimer()

	// Benchmark
	for i := 0; i < b.N; i += 1 {
		index := indices[r.rng.Intn(num_coords)]
		_, _ = rt.Get(index)
	}
}

// x-fast trie
func Benchmark_XFastTrie_Lookup(b *testing.B) {
	// setup
	r := NewBenchRNG(0)
	coords, cells := initBench(r)
	num_coords := len(coords)
	xft := xfastInsert(coords, cells)

	indices := make([]uint64, 0, num_coords)
	for _, cell := range cells {
		indices = append(indices, uint64(cell))
	}
	b.ResetTimer()

	// Benchmark
	for i := 0; i < b.N; i += 1 {
		index := indices[r.rng.Intn(num_coords)]
		_ = xft.Get(index)
	}
}

// y-fast trie
func Benchmark_YFastTrie_Lookup(b *testing.B) {
	// setup
	r := NewBenchRNG(0)
	coords, cells := initBench(r)
	num_coords := len(coords)
	yft := yfastInsert(coords, cells)

	indices := make([]uint64, 0, num_coords)
	for _, cell := range cells {
		indices = append(indices, uint64(cell))
	}
	b.ResetTimer()

	// Benchmark
	for i := 0; i < b.N; i += 1 {
		index := indices[r.rng.Intn(num_coords)]
		_ = yft.Get(index)
	}
}

