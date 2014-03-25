package succinctBitSet

import (
	"bytes"
	"fmt"
	"strconv"
)

const mask1 = word8Bit(0x55)
const mask2 = word8Bit(0x33)
const mask3 = word8Bit(0x0f)

var pascalRow8 = [...]uint64{1, 8, 28, 56, 70, 56, 28, 8, 1}
var pascalRow8Log2 = [...]uint64{0, 4, 5, 6, 7, 6, 5, 4, 0}

type BitSet struct {
	binomialLookup     []uint64
	binomialLookupLog2 []uint64
	bitcursor          uint64
	set                []uint64
	table              *table8Bit
	cLength            uint64
	blockCount         uint
	superBlockSize     uint
	bitSum             uint
	superBlocks        []superBlock
	blockLength        uint
}

type superBlock struct {
	offset  uint64
	rankSum uint
}

type table8Bit [9]row
type row []uint
type word8Bit uint8

// Count bits set (rank) from the most-significant up to a given
// position. Shamelessly taken from the excellent
// graphics.stanford.edu/~seander/bithacks.html
func (word word8Bit) popCountToBit(rank uint) uint {
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
	return uint(r)
}

func (word word8Bit) popCountAll() uint {
	return word.popCountToBit(8)
}

func New8BitTable() *table8Bit {
	table := new(table8Bit)
	for i := range pascalRow8 {
		table.addRow(i)
	}
	return table
}

func (table table8Bit) getOffset(popcount uint, block uint) int {
	for i, tableBitSet := range table[popcount] {
		if tableBitSet == block {
			return i
		}
	}
	return -1
}

func (table *table8Bit) addRow(i int) {
	table[i] = fixedPopCountBlocks(uint64(8), uint64(i))
}

func New() *BitSet {
	return &BitSet{
		binomialLookup:     pascalRow8[:],
		binomialLookupLog2: pascalRow8Log2[:],
		cLength:            4,
		bitcursor:          0,
		set:                make([]uint64, 1),
		table:              New8BitTable(),
		superBlockSize:     8,
		superBlocks:        make([]superBlock, 0),
		bitSum:             0,
		blockLength:        8,
	}
}

func (bitset BitSet) String() string {
	var outBuffer bytes.Buffer
	outBuffer.WriteRune('[')
	bitIndex := uint64(0)
	for blockIndex := uint(0); blockIndex < bitset.blockCount; blockIndex++ {
		outBuffer.Write([]byte("\033[32m"))
		class := bitset.getBits(bitIndex, uint64(bitset.cLength))
		fmt.Fprintf(&outBuffer, "%d", class)
		//fmt.Fprintf(&outBuffer, "%04b", class)

		offset := bitset.getBits(bitIndex+bitset.cLength, bitset.binomialLookupLog2[class])
		outBuffer.Write([]byte("\033[36m"))
		format := "\033[36m%0" + strconv.FormatInt(int64(bitset.binomialLookupLog2[class]), 10) + "b\033[39m%08b,"

		el := elementZero(class)
		for i := 0; i < int(offset); i++ {
			el = nextPerm(el)
		}

		fmt.Fprintf(&outBuffer, format, offset, el)
		bitIndex += bitset.cLength + bitset.binomialLookupLog2[class]
	}
	outBuffer.Bytes()[outBuffer.Len()-1] = '\033'
	outBuffer.Write([]byte("[39m"))
	outBuffer.WriteRune(']')
	return outBuffer.String()
}

func (bitset *BitSet) RecoverAsString() string {
	var outBuffer bytes.Buffer
	bitIndex := uint64(0)
	for blockIndex := uint(0); blockIndex < bitset.blockCount; blockIndex++ {
		class := bitset.getBits(bitIndex, bitset.cLength)
		offset := bitset.getBits(bitIndex+bitset.cLength, bitset.binomialLookupLog2[class])

		el := elementZero(class)
		for i := 0; i < int(offset); i++ {
			el = nextPerm(el)
		}

		fmt.Fprintf(&outBuffer, "%08b", el)
		bitIndex += bitset.cLength + bitset.binomialLookupLog2[class]
	}
	return outBuffer.String()
}

func (bitset *BitSet) AddFromBoolChan(bitChan <-chan bool) {
	blockLength := bitset.blockLength
	buffer := uint(0)

	i := uint(0)
	for bit := range bitChan {
		if bit {
			buffer = (1 << (blockLength - i%blockLength - 1)) | buffer
		}

		if (i+1)%blockLength == 0 {
			popcount := word8Bit(buffer).popCountAll()
			offset := bitset.table.getOffset(popcount, buffer)
			bitset.addBits(popcount, offset)
			buffer = 0
		}
		i++
	}
	popcount := word8Bit(buffer).popCountAll()
	offset := bitset.table.getOffset(popcount, buffer)
	bitset.addBits(popcount, offset)
}

func (bitset *BitSet) PrintSet() {
	// Write the already-set bits in blue
	fmt.Printf("\033[34m")
	for i := 0; i < len(bitset.set)-1; i++ {
		fmt.Printf("%064b", bitset.set[i])
	}

	// Write the rest of the bits in white
	finalSet := bitset.set[len(bitset.set)-1]
	if bitset.bitcursor == 0 {
		fmt.Printf("\033[39m")
	} else {
		var format bytes.Buffer
		fmt.Fprintf(&format, "%%0%db\033[39m", bitset.bitcursor)
		fmt.Printf(format.String(), finalSet>>(64-bitset.bitcursor))
	}

	if bitset.bitcursor >= 64 {
		fmt.Printf("  %d\n", bitset.bitcursor)
	} else {
		var format bytes.Buffer
		fmt.Fprintf(&format, "%%0%db\n", 64-bitset.bitcursor)
		fmt.Printf(format.String(), finalSet<<bitset.bitcursor)
	}
}

// Add a C, O pair (class, offset) to the bitset
func (bitset *BitSet) addBits(popCountClass uint, offset int) {
	// Append the popcount class bits
	if bitset.cLength+bitset.bitcursor <= 64 {
		bitset.set[len(bitset.set)-1] |= uint64(popCountClass) << (64 - bitset.cLength - bitset.bitcursor)
		bitset.bitcursor += bitset.cLength
	} else {
		remainder := (bitset.cLength + bitset.bitcursor) % 64
		bitset.set[len(bitset.set)-1] |= uint64(popCountClass) >> remainder
		bitset.set = append(bitset.set, uint64(popCountClass<<(64-remainder)))
		bitset.bitcursor = remainder
	}
	if bitset.bitcursor == 64 {
		bitset.set = append(bitset.set, 0)
		bitset.bitcursor = 0
	}
	bitset.bitSum += popCountClass

	// Append the offset bits
	bitSize := bitset.binomialLookupLog2[popCountClass]
	if bitSize+bitset.bitcursor <= uint64(64) {
		bitset.set[len(bitset.set)-1] |= uint64(offset) << (64 - bitSize - bitset.bitcursor)
		bitset.bitcursor += bitSize
	} else {
		remainder := (bitSize + bitset.bitcursor) % 64
		bitset.set[len(bitset.set)-1] |= uint64(offset) >> remainder
		bitset.set = append(bitset.set, 0)
		bitset.set[len(bitset.set)-1] |= uint64(popCountClass) << (64 - 2*bitSize - bitset.bitcursor + remainder)
		bitset.bitcursor = remainder
	}
	if bitset.bitcursor == 64 {
		bitset.set = append(bitset.set, 0)
		bitset.bitcursor = 0
	}
	bitset.blockCount++

	if bitset.blockCount%bitset.superBlockSize == 0 {
		bitset.superBlocks = append(bitset.superBlocks, superBlock{offset: uint64((len(bitset.set)-1)*64) + bitset.bitcursor, rankSum: bitset.bitSum})
	}
}

func (bitset BitSet) getBits(offset, n uint64) uint64 {
	subIndex := offset % 64
	// Does the requested bits flow into the next int64set?
	if (offset%64)+n > 64 {
		// if so, calculate the number of bits that overflow
		// Note that we're assuming that you'll never be
		// reqeusting more than 64 bits.
		remainder := (offset % 64) + n - 64
		return bitset.set[offset/64]&((1<<(64-subIndex))-1)<<remainder | bitset.set[offset/64+1]>>(64-remainder)
	} else {
		return bitset.set[offset/64] >> (64 - subIndex - n) & ((1 << n) - 1)
	}
}

func (bitset *BitSet) Rank(ith uint) uint {
	count := uint(0)
	bitIndex := uint64(0)

	// Which block contains our bit of interest?
	var targetBlockIndex uint
	if ith/bitset.blockLength < bitset.blockCount {
		targetBlockIndex = ith / bitset.blockLength
	} else {
		targetBlockIndex = bitset.blockCount
	}

	//Which superblock precedes our block of interest?
	superBlockIndex := targetBlockIndex / bitset.superBlockSize
	if superBlockIndex > 0 {
		superBlock := bitset.superBlocks[superBlockIndex-1]
		count = superBlock.rankSum
		bitIndex += superBlock.offset
	}

	for blockIndex := uint(0); blockIndex < targetBlockIndex-superBlockIndex*bitset.superBlockSize; blockIndex++ {
		class := bitset.getBits(bitIndex, bitset.cLength)
		count += uint(class)
		bitIndex += bitset.cLength + bitset.binomialLookupLog2[class]
	}

	finalClass := bitset.getBits(bitIndex, bitset.cLength)
	finalOffset := bitset.getBits(bitIndex+bitset.cLength, bitset.binomialLookupLog2[finalClass])

	el := elementZero(finalClass)
	for i := 0; i < int(finalOffset); i++ {
		el = nextPerm(el)
	}

	count += word8Bit(el).popCountToBit(uint(ith % bitset.blockLength))
	return count
}

// Taken from http://graphics.stanford.edu/~seander/bithacks.html
func nextPerm(v uint) uint {
	t := (v | (v - 1)) + 1
	w := t | ((((t & -t) / (v & -v)) >> 1) - 1)
	return w
}

// Generates the first permutation with a given number of set bits b.
func elementZero(c uint64) uint {
	return (1 << c) - 1
}

func fixedPopCountBlocks(b, p uint64) row {
	blocks := make([]uint, pascalRow8[p])
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
