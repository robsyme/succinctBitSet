package succinctBitSet

import (
	"fmt"
)

const mask1 = word8Bit(0x55)
const mask2 = word8Bit(0x33)
const mask3 = word8Bit(0x0f)

var pascalRow8 = [...]uint32{1, 8, 28, 56, 70, 56, 28, 8, 1}
var pascalRow8Log2 = [...]uint32{0, 4, 5, 6, 7, 6, 5, 4, 0}

type BitSet struct {
	binomialLookup     []uint32
	binomialLookupLog2 []uint32
	bitcursor          uint32
	set                []uint64
	table              Table
	cLength            uint32
}

type Table interface {
	getOffset(popcount int, block uint32) int
	blockLength() uint32
	addRow(uint32)
}

type Block interface {
	popCountToBit(int) int
}

type table8Bit [9]row
type row []uint32
type row8Bit []uint32
type word8Bit uint8
type classOffsetPair [2]int32

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

func (word word8Bit) popCountAll() int {
	return word.popCountToBit(8)
}

func New8BitTable() *table8Bit {
	table := new(table8Bit)
	for i := range pascalRow8 {
		table.addRow(uint32(i))
	}
	return table
}

func (table table8Bit) getOffset(popcount int, block uint32) int {
	for i, tableBitSet := range table[popcount] {
		if tableBitSet == block {
			return i
		}
	}
	return -1
}

func (table *table8Bit) blockLength() uint32 {
	return 8
}

func (table *table8Bit) addRow(i uint32) {
	table[i] = fixedPopCountBlocks(uint32(table.blockLength()), i)
}

func New8BitSet() BitSet {
	return BitSet{
		binomialLookup:     pascalRow8[:],
		binomialLookupLog2: pascalRow8Log2[:],
		cLength:            3,
		bitcursor:          0,
		set:                make([]uint64, 1),
		table:              New8BitTable(),
	}
}

func (bitset *BitSet) AddFromBoolChan(bitChan <-chan bool) {
	blockLength := bitset.table.blockLength()
	buffer := 0
	i := uint32(0)
	for bit := range bitChan {
		if (i+1)%blockLength != 0 {
			if bit {
				tmp := 1 << (i - 1)
				buffer = buffer | tmp
			}
		} else { //Add the new word to the bit string
			//fmt.Printf("Adding: %08b\n", buffer)
			popcount := word8Bit(buffer).popCountAll()
			offset := bitset.table.getOffset(popcount, uint32(buffer))
			bitset.addBits(popcount, offset)
			i = 0
			buffer = 0
		}
		i++
	}
}

// Add a C, O pair (class, offset) to the bitset
func (bitset *BitSet) addBits(popCountClass, offset int) {
	// Append the popcount class bits
	if bitset.cLength+bitset.bitcursor <= 64 {
		bitset.set[len(bitset.set)-1] |= uint64(popCountClass) << (64 - bitset.cLength - bitset.bitcursor)
		bitset.bitcursor += bitset.cLength
	} else {
		remainder := (bitset.cLength + bitset.bitcursor) % 64
		bitset.set[len(bitset.set)-1] |= uint64(offset) >> remainder
		bitset.set = append(bitset.set, 0)
		bitset.set[len(bitset.set)-1] |= uint64(popCountClass) << (64 - 2*bitset.cLength - bitset.bitcursor + remainder)
		bitset.bitcursor = remainder
	}

	// Append the offset bits
	bitSize := bitset.binomialLookupLog2[popCountClass]
	if bitSize+bitset.bitcursor <= 64 {
		bitset.set[len(bitset.set)-1] |= uint64(offset) << (64 - bitSize - bitset.bitcursor)
		bitset.bitcursor += bitSize
	} else {
		remainder := (bitSize + bitset.bitcursor) % 64
		bitset.set[len(bitset.set)-1] |= uint64(offset) >> remainder
		bitset.set = append(bitset.set, 0)
		bitset.set[len(bitset.set)-1] |= uint64(popCountClass) << (64 - 2*bitSize - bitset.bitcursor + remainder)
		bitset.bitcursor = remainder
	}
}

func (bitset *BitSet) Rank(ith uint32) int {
	var currentInt64 uint64
	var class uint64
	targetBlockGlobal := int((ith) / bitset.table.blockLength())

	count := 0
	blockIndex := 0
	bitIndex := uint32(0)
	for blockIndex < targetBlockGlobal {
		fmt.Printf("%d (blockIndex) < %d (targetBlockGlobal)\n", blockIndex, targetBlockGlobal)
		setIndex := bitIndex / 64
		currentInt64 = bitset.set[setIndex]
		for bitIndex-setIndex*64 < 63 && blockIndex < targetBlockGlobal {
			class = currentInt64 >> (64 - bitset.cLength - bitIndex%64) & ((1 << bitset.cLength) - 1)
			format := " "
			for i := 0; i < int(setIndex); i++ {
				format += "                                                                 "
			}
			for i := 0; i < int(bitIndex%64); i++ {
				format += " "
			}
			format += "..."
			for i := 0; i < int(bitset.binomialLookupLog2[class]); i++ {
				format += "_"
			}
			fmt.Println(format)
			fmt.Printf("%b\n", bitset.set)
			bitIndex += bitset.binomialLookupLog2[class] + bitset.cLength
			count += int(class)
			blockIndex++
		}
	}

	currentInt64 = bitset.set[bitIndex/64]
	class = currentInt64 >> (64 - bitset.cLength - bitIndex) & ((1 << bitset.cLength) - 1)
	offset := currentInt64 >> (64 - bitset.cLength - bitIndex - bitset.binomialLookupLog2[class]) & ((1 << bitset.binomialLookupLog2[class]) - 1)

	el := elementZero(uint32(class))
	for i := 0; i < int(offset); i++ {
		el = nextPerm(el)
	}

	count += word8Bit(el).popCountToBit(uint(ith % bitset.table.blockLength()))
	return count
}

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
