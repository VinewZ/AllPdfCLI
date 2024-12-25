package main

import (
	"os"
	"sort"
)


func ByNumber(entries []os.DirEntry, extractor func(os.DirEntry) int) sort.Interface {
	return &byNumber{
		entries:   entries,
		extractor: extractor,
	}
}

type byNumber struct {
	entries   []os.DirEntry
	extractor func(os.DirEntry) int
}

func (b *byNumber) Len() int {
	return len(b.entries)
}

func (b *byNumber) Less(i, j int) bool {
	return b.extractor(b.entries[i]) < b.extractor(b.entries[j])
}

func (b *byNumber) Swap(i, j int) {
	b.entries[i], b.entries[j] = b.entries[j], b.entries[i]
}
