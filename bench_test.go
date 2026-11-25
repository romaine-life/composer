package composer

import (
	"strings"
	"testing"
)

// draft is a 2000-character message of words, as a long message being written is,
// with the cursor at its end.
func draft() *Box {
	b := New(10)
	b.SetWidth(100)
	b.SetAtoms(marked)
	b.SetValue(strings.Repeat("a few words ⟦tmux.window⟧ ", 80)[:2000])
	return b
}

// BenchmarkHeight is one Height call on a long draft: at-claude asks it many times
// a frame.
func BenchmarkHeight(b *testing.B) {
	box := draft()
	for b.Loop() {
		box.Height()
	}
}

// BenchmarkView is one View of a long draft.
func BenchmarkView(b *testing.B) {
	box := draft()
	for b.Loop() {
		box.View(Styles{}, true)
	}
}

// BenchmarkTypeAndView is a key typed into a long draft and the box shown after it.
func BenchmarkTypeAndView(b *testing.B) {
	box := draft()
	for b.Loop() {
		box.Insert("x")
		box.View(Styles{}, true)
	}
}
