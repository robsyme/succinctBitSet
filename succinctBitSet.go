package succinctBitSet

import (
	"fmt"
)

const mask1 = word8Bit(0x55)
const mask2 = word8Bit(0x33)
const mask3 = word8Bit(0x0f)

var pascalRow8 = [...]uint32{1, 8, 28, 56, 70, 56, 28, 8, 1}

type BitSet struct {
	blockcursor int64
	bitcursor   uint
	set         []uint32
	table       Table
}

type Table interface {
	getOffset(popcount int, block uint32) int
	blockLength() uint
}

type Block interface {
	popCountToBit(int) int
}

type table8Bit [9]row
type row []uint32
type row8Bit []uint32
type word8Bit uint8

// Count bits set (rank) from the most-significant up to a given
// position. Shamelessley taken from the excellent
// graphics.stanford.edu/~seander/bithacks.html
func (word word8Bit) popCountToBit(rank uint) int {
	var r word8Bit
	if rank >= 8 {
		r = word
	} else {
		r = word >> (8 - rank)
	}

	r = r - ((r >> 1) & mask1)
	r = (r & mask2) + ((r >> 2) & mask2)
	r = (r + (r >> 4)) & mask3
	r = r % 255
	return int(r)
}

func (table *table8Bit) getOffset(popcount int, block uint32) int {
	for i, tableBitSet := range table[popcount] {
		if tableBitSet == block {
			return i
		}
	}
	return -1
}

func (table *table8Bit) blockLength() uint {
	return 8
}

func (table *table8Bit) addRow(i uint32) {
	table[i] = fixedPopCountBlocks(uint32(table.blockLength()), i)
}

func New8BitSet() BitSet {
	table := new(table8Bit)
	for i := range pascalRow8 {
		table.addRow(uint32(i))
	}
	return BitSet{table: table, blockcursor: 0, bitcursor: 0}
}

func (bitset *BitSet) AddFromBoolChan(bitChan <-chan bool) {
	blockLength := bitset.table.blockLength()
	buffer := 0
	for bit := range bitChan {
		if (bitset.bitcursor+1)%blockLength != 0 {
			if bit {
				tmp := 1 << (bitset.bitcursor - 1)
				buffer = buffer | tmp
			}
		} else {
			fmt.Printf("%08b (%d)\n", buffer, buffer)
			bitset.bitcursor = 0
			buffer = 0
		}
		bitset.bitcursor++
	}
}

// Generates the next permutation with a given amount of set bits b.
// Taken from http://graphics.stanford.edu/~seander/bithacks.html
func nextPerm(v uint32) uint32 {
	t := (v | (v - 1)) + 1
	w := t | ((((t & -t) / (v & -v)) >> 1) - 1)
	return w
}

// Generates the first permutation with a given number of set bits b.
func elementZero(c uint32) uint32 {
	return (1 << c) - 1
}

func fixedPopCountBlocks(b, p uint32) row {
	blocks := make([]uint32, pascalRow8[p])
	if p == 0 {
		blocks[0] = 0
	} else {
		v := elementZero(p)
		blockMask := elementZero(b)
		for i := range blocks {
			blocks[i] = v
			v = nextPerm(v) & blockMask
		}
	}
	return blocks
}
