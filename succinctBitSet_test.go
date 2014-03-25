package succinctBitSet

import (
	. "github.com/smartystreets/goconvey/convey"
	"math/rand"
	"testing"
)

func TestTableCreation(t *testing.T) {
	Convey("Given a basic BitSet with block length 8", t, func() {
		bitset := New()
		Convey("It should not be nil", func() {
			So(bitset, ShouldNotBeNil)
		})

		Convey("It should have a set of length 1", func() {
			So(len(bitset.set), ShouldEqual, 1)
		})

		Convey("It should have a table", func() {
			table := bitset.table

			Convey("which has a block length of 8 bits", func() {
				So(table.blockLength(), ShouldEqual, uint32(8))
			})

			Convey("which can calculate the offset for a given bitset", func() {
				So(table.getOffset(0, 0), ShouldEqual, 0)
				So(table.getOffset(1, 1), ShouldEqual, 0)
				So(table.getOffset(1, 2), ShouldEqual, 1)
				So(table.getOffset(2, 3), ShouldEqual, 0)
				So(table.getOffset(1, 4), ShouldEqual, 2)
				So(table.getOffset(2, 5), ShouldEqual, 1)
				So(table.getOffset(2, 6), ShouldEqual, 2)
				So(table.getOffset(1, 32), ShouldEqual, 5)
			})

		})

		Convey("It should take a channel of boolean bits", func() {
			sum := 0
			wordCount := 20
			bits := make(chan bool, 8*wordCount)
			go func() {
				r := rand.New(rand.NewSource(100))
				for i := 0; i < 8*wordCount; i++ {
					if r.Int()%2 == 0 {
						sum++
						bits <- true
					} else {
						bits <- false
					}
				}
				close(bits)
			}()
			bitset.AddFromBoolChan(bits)
		})

		Convey("It should be able to add bits to the bitset", func() {
			Convey("It should be answer basic Rank queries", func() {
				So(bitset.Rank(0), ShouldEqual, 0)
				So(bitset.Rank(1), ShouldEqual, 1)
				So(bitset.Rank(2), ShouldEqual, 2)
				So(bitset.Rank(3), ShouldEqual, 2)
				So(bitset.Rank(8), ShouldEqual, 6)
				So(bitset.Rank(9), ShouldEqual, 6)
				So(bitset.Rank(11), ShouldEqual, 7)
			})
			Convey("Rank should cross the 64-bit barrier without problem", func() {
				So(bitset.Rank(100), ShouldEqual, 57)
			})
			Convey("Rank queries larger than the set should not fail", func() {
				So(bitset.Rank(300), ShouldEqual, 90)
			})
		})
	})

	Convey("Given an 8-bit word '01001010'", t, func() {
		w := word8Bit(74)
		Convey("it can calculate the popCount to each offset", func() {
			So(w.popCountToBit(0), ShouldEqual, 0)
			So(w.popCountToBit(1), ShouldEqual, 0)
			So(w.popCountToBit(2), ShouldEqual, 1)
			So(w.popCountToBit(3), ShouldEqual, 1)
			So(w.popCountToBit(4), ShouldEqual, 1)
			So(w.popCountToBit(5), ShouldEqual, 2)
			So(w.popCountToBit(6), ShouldEqual, 2)
			So(w.popCountToBit(7), ShouldEqual, 3)
			So(w.popCountToBit(8), ShouldEqual, 3)
			So(w.popCountToBit(9), ShouldEqual, 3)
		})
	})
	Convey("Given an 8-bit word '11111111'", t, func() {
		w := word8Bit(255)
		Convey("it can calculate the popCount to each offset", func() {
			So(w.popCountToBit(0), ShouldEqual, 0)
			So(w.popCountToBit(1), ShouldEqual, 1)
			So(w.popCountToBit(2), ShouldEqual, 2)
			So(w.popCountToBit(3), ShouldEqual, 3)
			So(w.popCountToBit(4), ShouldEqual, 4)
			So(w.popCountToBit(5), ShouldEqual, 5)
			So(w.popCountToBit(6), ShouldEqual, 6)
			So(w.popCountToBit(7), ShouldEqual, 7)
			So(w.popCountToBit(8), ShouldEqual, 8)
			So(w.popCountToBit(9), ShouldEqual, 8)
		})
	})

	Convey("A blank 8-bit bitset should be encoded and decoded correctly", t, func() {
		input := "00000000"
		bitset := New()
		bits := make(chan bool)
		go func() {
			for _, c := range input {
				if c == '1' {
					bits <- true
				} else {
					bits <- false
				}
			}
			close(bits)
		}()
		bitset.AddFromBoolChan(bits)
		So(bitset.RecoverAsString(), ShouldStartWith, input)
	})

	Convey("The bitset 000000001 should be encoded and decoded correctly", t, func() {
		input := "000000001"
		bitset := New()
		bits := make(chan bool)
		go func() {
			for _, c := range input {
				if c == '1' {
					bits <- true
				} else {
					bits <- false
				}
			}
			close(bits)
		}()
		bitset.AddFromBoolChan(bits)
		So(bitset.RecoverAsString(), ShouldStartWith, input)
	})

	Convey("Index should not fail when setting popCount up to 64-bit border", t, func() {
		input := "1111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111101"
		bitset := New()
		bits := make(chan bool)
		go func() {
			for _, c := range input {
				if c == '1' {
					bits <- true
				} else {
					bits <- false
				}
			}
			close(bits)
		}()
		bitset.AddFromBoolChan(bits)
		So(bitset.RecoverAsString(), ShouldStartWith, input)
	})
	Convey("Index should not fail when setting offset up to 64-bit border", t, func() {
		input := "11111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111011111111"
		bitset := New()
		bits := make(chan bool)
		go func() {
			for _, c := range input {
				if c == '1' {
					bits <- true
				} else {
					bits <- false
				}
			}
			close(bits)
		}()
		bitset.AddFromBoolChan(bits)
		So(bitset.RecoverAsString(), ShouldStartWith, input)
	})

	Convey("Given a bitset with 96 0s then 96 1s", t, func() {
		input := "00000000000000000000000000000000000000000000000000000000000000001111111111111111111111111111111111111111111111111111111111111111"
		bitset := New()
		bits := make(chan bool)
		go func() {
			for _, c := range input {
				if c == '1' {
					bits <- true
				} else {
					bits <- false
				}
			}
			close(bits)
		}()
		bitset.AddFromBoolChan(bits)

		Convey("An array of superblocks should be present", func() {
			So(bitset.superBlocks, ShouldNotBeNil)
		})

		Convey("The first superblock should point to bit number 32", func() {
			So(bitset.superBlocks[0].rankSum, ShouldEqual, 0)
			So(bitset.superBlocks[0].offset, ShouldEqual, 32)
		})
		Convey("The second superblock should point to bit number 64", func() {
			So(bitset.superBlocks[1].rankSum, ShouldEqual, 64)
			So(bitset.superBlocks[1].offset, ShouldEqual, 64)
		})
	})

	Convey("Given a random bitstring", t, func() {
		input := "110011110101011100101111101000011111011100000101000001111110001110010001111010111001000111101011011010111111010100101011100100111100011111101101000010100011110011100111"
		bitset := New()
		bits := make(chan bool)
		go func() {
			for _, c := range input {
				if c == '1' {
					bits <- true
				} else {
					bits <- false
				}
			}
			close(bits)
		}()
		bitset.AddFromBoolChan(bits)

		Convey("There should be two superblocks", func() {
			So(len(bitset.superBlocks), ShouldEqual, 2)
		})

		Convey("The first superblock should have counted 36 bits in the first 76 bases", func() {
			So(bitset.superBlocks[0].offset, ShouldEqual, 76)
			So(bitset.superBlocks[0].rankSum, ShouldEqual, 36)
		})

		Convey("The second superblock should have counted 73 bits in the first 155 bases", func() {
			So(bitset.superBlocks[1].offset, ShouldEqual, 155)
			So(bitset.superBlocks[1].rankSum, ShouldEqual, 73)
		})

		Convey("Calculating rank before the first superblock", func() {
			So(bitset.Rank(10), ShouldEqual, 7)
			So(bitset.Rank(27), ShouldEqual, 18)
		})

		Convey("Calculating rank after the first superblock", func() {
			So(bitset.Rank(65), ShouldEqual, 37)
		})

	})
}
