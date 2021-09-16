//+build linux_bpf

package kprobe

import (
	"bytes"
	"unsafe"

	"github.com/DataDog/datadog-agent/pkg/network"
	netebpf "github.com/DataDog/datadog-agent/pkg/network/ebpf"
	"github.com/DataDog/ebpf"
)

type TagsSet struct {
	m            *ebpf.Map
	set          map[string]uint32
	nextTagValue uint32
}

func newTagsSet(m *ebpf.Map) *TagsSet {
	return &TagsSet{
		m:            m,
		set:          make(map[string]uint32),
		nextTagValue: uint32(len(netebpf.StaticTagsStrings)),
	}
}

func (ts *TagsSet) Tag(conn *network.ConnectionStats, tag string) {
	conn.Tags = append(conn.Tags, ts.addTag(tag))
}

func (ts *TagsSet) getTagsStrings() (strs network.Tags) {
	max := uint32(len(netebpf.StaticTagsStrings) - 1)
	rset := make(map[uint32]string)
	for k, v := range ts.set {
		if v > max {
			max = v
		}
		rset[v] = k
	}
	for i, t := range netebpf.StaticTagsStrings {
		rset[uint32(i)] = t
	}

	for i := uint32(0); i <= max; i++ {
		v, found := rset[i]
		if !found {
			/* we should never hit this */
			continue
		}
		strs = append(strs, v)
	}
	return strs
}

func (ts *TagsSet) addTag(tag string) (v uint32) {
	if v, found := ts.set[tag]; found {
		return v
	}
	v = ts.nextTagValue
	ts.set[tag] = v
	ts.nextTagValue++
	return v
}

// getIndexes lookup on ConnTuple, update the internal map and return the tags indexes
func (ts *TagsSet) getIndexes(tuple *netebpf.ConnTuple) (uintTags []uint32) {
	tags := new(netebpf.Tags)
	err := ts.m.Lookup(unsafe.Pointer(tuple), unsafe.Pointer(tags))
	if err == nil {
		return
	}

	tagsValues := []byte{}
	for _, v := range tags.Value {
		tagsValues = append(tagsValues, byte(v))
	}
	// split the multiple tags, thanks to the tailing \0
	for _, t := range bytes.Split(tagsValues, []byte{0}) {
		if len(t) == 0 {
			continue
		}
		uintTags = append(uintTags, ts.addTag(string(t)))
	}
	return uintTags
}

func getStaticTagsFromConnStats(s *netebpf.ConnStats) (r []uint32) {
	var v uint64 = s.Tags
	for i := 0; i < 8; /*sizeof(uint64)/sizeof(uint8)*/ i++ {
		o := i * 8
		tag := uint32((v >> o) & 0xff)
		if tag > uint32(0) {
			r = append(r, tag)
		}
	}
	return r
}
