package succinctBitSet

import ()

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
	addRow(int)
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
// position. Shamelessly taken from the excellent
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
		table.addRow(i)
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

func (table *table8Bit) addRow(i int) {
	table[i] = fixedPopCountBlocks(uint64(table.blockLength()), uint64(i))
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

func (bitset BitSet) getBits(offset, n uint32) uint64 {
	subIndex := offset % 64
	// Does the requested bits flow into the next int64set?
	if (offset%64)+n > 64 {
		// if so, calculate the number of bits that overflow
		// Note that we're assuming that you'll never be
		// reqeusting more than 64 bits.
		remainder := (offset % 64) + n - 64
		buffer := bitset.set[offset/64]&((1<<(64-subIndex))-1)<<remainder | bitset.set[offset/64+1]>>(64-remainder)
		return buffer
	} else {
		//Shift 'offset' bits to the right then mask 'n' bits
		return bitset.set[offset/64] >> (64 - subIndex - n) & ((1 << n) - 1)
	}
}

func (bitset *BitSet) Rank(ith uint32) int {
	targetBlockGlobal := (ith) / bitset.table.blockLength()

	count := 0
	bitIndex := uint32(0)
	for i := uint32(0); i < targetBlockGlobal; i++ {
		class := bitset.getBits(bitIndex, bitset.cLength)
		offset := bitset.getBits(bitIndex+bitset.cLength, bitset.binomialLookupLog2[class])

		el := elementZero(class)
		for i := 0; i < int(offset); i++ {
			el = nextPerm(el)
		}

		count += int(class)
		bitIndex += bitset.cLength + bitset.binomialLookupLog2[class]
	}

	finalClass := bitset.getBits(bitIndex, bitset.cLength)
	finalOffset := bitset.getBits(bitIndex+bitset.cLength, bitset.binomialLookupLog2[finalClass])

	el := elementZero(finalClass)
	for i := 0; i < int(finalOffset); i++ {
		el = nextPerm(el)
	}

	count += word8Bit(el).popCountToBit(uint(ith % bitset.table.blockLength()))
	return int(count)
}

// Taken from http://graphics.stanford.edu/~seander/bithacks.html
func nextPerm(v uint32) uint32 {
	t := (v | (v - 1)) + 1
	w := t | ((((t & -t) / (v & -v)) >> 1) - 1)
	return w
}

// Generates the first permutation with a given number of set bits b.
func elementZero(c uint64) uint32 {
	return (1 << c) - 1
}

func fixedPopCountBlocks(b, p uint64) row {
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
