package succinctBitSet

import (
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"math/rand"
	"testing"
)

func TestTableCreation(t *testing.T) {
	Convey("Given a basic BitSet with block length 8", t, func() {
		bitset := New8BitSet()
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
			wordCount := 20
			bits := make(chan bool, 8*wordCount)
			go func() {
				r := rand.New(rand.NewSource(100))
				for i := 0; i < 8*wordCount; i++ {
					if r.Int()%2 == 0 {
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
				So(bitset.Rank(100), ShouldEqual, 51)
			})
			Convey("Rank queries larger than the set should not fail", func() {
				fmt.Println(&bitset)
				So(bitset.Rank(300), ShouldEqual, 79)
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
}
