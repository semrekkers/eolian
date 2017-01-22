package module

import "sort"

func newMarkers() *markers {
	return &markers{
		indexes: []int{0},
	}
}

type markers struct {
	indexes []int
}

func (b *markers) Create(i int) {
	b.indexes = append(b.indexes, i)
	sort.Sort(&indexSorter{b.indexes})
}

func (b *markers) Count() int {
	return len(b.indexes)
}

func (b *markers) At(i int) int {
	return b.indexes[i]
}

func (b *markers) Erase(end int) {
	if end == len(b.indexes)-1 {
		return
	}
	b.indexes = append(b.indexes[:end], b.indexes[end+1:]...)
}

func (b *markers) GetRange(organize Value) (int, int) {
	size := len(b.indexes)
	if size == 2 {
		return 0, size - 1
	}
	zoneSize := 1 / float64(size-1)
	start := minInt(size-2, int(float64(organize)/zoneSize))
	end := minInt(size-1, start+1)
	return start, end
}

type indexSorter struct {
	indexes []int
}

func (s *indexSorter) Len() int           { return len(s.indexes) }
func (s *indexSorter) Less(i, j int) bool { return s.indexes[i] < s.indexes[j] }
func (s *indexSorter) Swap(i, j int) {
	s.indexes[i], s.indexes[j] = s.indexes[j], s.indexes[i]
}
