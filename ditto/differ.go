// Copyright 2025 DittoLive Incorporated. All rights reserved.

package ditto

import (
	"fmt"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/getditto/ditto-go-sdk/v5/internal/cbor"
	"github.com/getditto/ditto-go-sdk/v5/internal/ffi"
)

// Move represents a move of an item from a particular index in the old
// array to a particular index in the new array.
type Move struct {
	// From is the index in the old array from which the item has been moved.
	From int `json:"from" cbor:"from"`
	// To is the index in the new array to which the item has been moved.
	To int `json:"to" cbor:"to"`
}

// String returns a string representation of the Move.
//
// This is intended for use in logging and debugging.
func (m Move) String() string {
	return fmt.Sprintf("{from:%d to:%d}", m.From, m.To)
}

// Diff represents a diff between two arrays.
//
// Create a diff between arrays of QueryResultItem using a Differ.
type Diff struct {
	// Insertions is the set of indexes in the new array at which new items have been inserted.
	Insertions []int `json:"insertions" cbor:"insertions"`
	// Deletions is the set of indexes in the old array at which old items have been deleted.
	Deletions []int `json:"deletions" cbor:"deletions"`
	// Updates is the set of indexes in the new array at which items have been updated.
	Updates []int `json:"updates" cbor:"updates"`
	// Moves is a set of tuples each representing a move of an item from a particular index in the old
	// array to a particular index in the new array.
	Moves []Move `json:"moves" cbor:"moves"`
}

// String returns a string representation of the Diff.
//
// This is intended for use in logging and debugging.
func (d *Diff) String() string {
	if d == nil {
		return "<nil>"
	}

	var b strings.Builder
	b.WriteByte('{')

	needSpace := false

	if len(d.Insertions) > 0 {
		fmt.Fprintf(&b, "insertions:%v", d.Insertions)
		needSpace = true
	}

	if len(d.Deletions) > 0 {
		if needSpace {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "deletions:%v", d.Deletions)
		needSpace = true
	}

	if len(d.Updates) > 0 {
		if needSpace {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "updates:%v", d.Updates)
		needSpace = true
	}

	if len(d.Moves) > 0 {
		if needSpace {
			b.WriteByte(' ')
		}
		b.WriteString("moves:[")
		for i, m := range d.Moves {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(m.String())
		}
		b.WriteByte(']')
	}

	b.WriteByte('}')
	return b.String()
}

// Equal returns true if this Diff is equal to another Diff.
func (d *Diff) Equal(other *Diff) bool {
	if d == nil && other == nil {
		return true
	}
	if d == nil || other == nil {
		return false
	}

	return slices.Equal(d.Insertions, other.Insertions) &&
		slices.Equal(d.Deletions, other.Deletions) &&
		slices.Equal(d.Updates, other.Updates) &&
		slices.Equal(d.Moves, other.Moves)
}

// Differ calculates diffs between arrays of QueryResultItem.
//
// Use a Differ with a StoreObserver to get the diff between subsequent query
// results delivered by the store observer.
type Differ struct {
	mu     sync.Mutex
	handle *ffi.DifferHandle
}

// NewDiffer creates a new Differ.
//
// The caller should call Close on the Differ when it is no longer needed, to free the memory it uses.
func NewDiffer() *Differ {
	handle := ffi.NewDiffer()
	if handle == nil {
		return nil
	}
	differ := &Differ{handle: handle}

	// TODO: Should setting the finalizer be handled in the internal.ffi package?
	runtime.SetFinalizer(
		differ, func(d *Differ) {
			d.mu.Lock()
			defer d.mu.Unlock()
			if d.handle != nil {
				d.handle.Free()
				d.handle = nil
			}
		},
	)

	return differ
}

// DiffResult calculates the diff of the provided QueryResult against the last set of items that were passed to
// this differ.
//
// The returned Diff identifies changes from the old array of items to the new array
// of items using indices into both arrays.
//
// Initially, the Differ has no items, so the first call to this method will always return a
// Diff showing all items as insertions.
//
// The identity of items is determined by their _id field.
func (d *Differ) DiffResult(result *QueryResult) *Diff {
	items := result.itemsAsSlice()

	// Note: DiffItems will lock the mutex
	diff := d.DiffItems(items)

	for _, item := range items {
		item.Close()
	}

	return diff
}

// DiffItems calculates the diff of the provided items against the last set of items that were passed to
// this differ.
//
// The returned Diff identifies changes from the old array of items to the new array
// of items using indices into both arrays.
//
// Initially, the Differ has no items, so the first call to this method will always return a
// diff showing all items as insertions.
//
// The identity of items is determined by their _id field.
//
// DiffItems does not close any of the items passed into it.
//
// If you have a QueryResult, use DiffResult instead of this method, to avoid the need to manage
// the lifetimes of the items.
func (d *Differ) DiffItems(items []*QueryResultItem) *Diff {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.handle == nil {
		LogError("gosdk: Diff: handle is nil")
		return &Diff{}
	}

	// Convert QueryResultItem to FFI handles
	ffiItems := make([]*ffi.QueryResultItemHandle, len(items))
	for i, item := range items {
		if item != nil && item.handle != nil {
			ffiItems[i] = item.handle
		} else {
			LogErrorF("gosdk: Diff: item at index %d is nil", i)
		}
	}

	diffBytes, err := d.handle.DiffItems(ffiItems)
	if err != nil {
		LogErrorF("gosdk: Diff: FFI returned error: %v", err)
		return &Diff{}
	}

	// Decode the CBOR bytes to create our Diff struct
	// The FFI returns moves as arrays [from, to], not as objects,
	// so we need to decode to an intermediate structure first
	var rawDiff struct {
		Insertions []int   `cbor:"insertions"`
		Deletions  []int   `cbor:"deletions"`
		Updates    []int   `cbor:"updates"`
		Moves      [][]int `cbor:"moves"` // Array of [from, to] pairs
	}

	err = cbor.Decode(diffBytes, &rawDiff)
	if err != nil {
		LogErrorF("gosdk: Diff: CBOR deserialization returned error: %v", err)
		return &Diff{}
	}

	// Convert the raw diff to our Diff type
	// Use nil for empty slices
	diff := Diff{}
	if len(rawDiff.Insertions) > 0 {
		diff.Insertions = rawDiff.Insertions
	}
	if len(rawDiff.Deletions) > 0 {
		diff.Deletions = rawDiff.Deletions
	}
	if len(rawDiff.Updates) > 0 {
		diff.Updates = rawDiff.Updates
	}
	if len(rawDiff.Moves) > 0 {
		diff.Moves = make([]Move, 0, len(rawDiff.Moves))
		for _, move := range rawDiff.Moves {
			if len(move) == 2 {
				diff.Moves = append(
					diff.Moves, Move{
						From: move[0],
						To:   move[1],
					},
				)
			}
		}
	}

	return &diff
}

// Close releases all resources associated with the Differ.
//
// After calling Close, the Differ should not be used.
func (d *Differ) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.handle != nil {
		// Clear the finalizer since we're closing manually
		runtime.SetFinalizer(d, nil)
		d.handle.Free()
		d.handle = nil
	}
}
