package composer

import (
	"reflect"
	"testing"

	"github.com/gdamore/tcell/v2"
)

// TestTheLayoutFollowsEveryChange lays the box out before each change, so the kept
// layout is the old one, and checks it after against a box built new in the same
// state, which has nothing kept: a change that left the layout stale shows here.
func TestTheLayoutFollowsEveryChange(t *testing.T) {
	b := New(4)
	b.SetWidth(12)
	b.SetAtoms(marked)
	edit := func(name string, do func()) struct {
		name string
		do   func()
		edit bool
	} {
		return struct {
			name string
			do   func()
			edit bool
		}{name, do, true}
	}
	steps := []struct {
		name string
		do   func()
		edit bool // It changes the text.
	}{
		edit("type", func() { typed(t, b, "some words that wrap ⟦tmux.window⟧ here") }),
		edit("new line", func() { b.Do(NewLine) }),
		edit("type again", func() { typed(t, b, "second line") }),
		edit("backspace", func() { press(b, tcell.KeyBackspace2, 0) }),
		edit("delete word back", func() { press(b, tcell.KeyCtrlW, 0) }),
		edit("backspace joins lines", func() { b.SetPosition(1, 0); press(b, tcell.KeyBackspace2, 0) }),
		edit("delete", func() { b.SetPosition(0, 0); press(b, tcell.KeyDelete, 0) }),
		edit("delete word forward", func() { press(b, tcell.KeyDelete, tcell.ModCtrl) }),
		edit("delete to end", func() { b.SetPosition(0, 6); press(b, tcell.KeyCtrlK, 0) }),
		edit("delete to start", func() { press(b, tcell.KeyCtrlU, 0) }),
		edit("paste", func() { b.Insert("pasted ⟦a.b⟧ text\nover two lines that wrap") }),
		edit("replace", func() { b.Replace(0, 0, 3, "⟦tmux.pane⟧") }),
		edit("undo", func() { b.Undo() }),
		edit("undo again", func() { b.Undo() }),
		edit("redo", func() { b.Redo() }),
		{"set width", func() { b.SetPosition(0, 0); b.SetWidth(7) }, false},
		{"atoms off", func() { b.SetAtoms(nil) }, false},
		{"atoms on", func() { b.SetAtoms(marked) }, false},
		edit("set value", func() { b.SetValue("a new ⟦draft.one⟧ brought back\nwith lines") }),
		{"width again", func() { b.SetWidth(30) }, false},
		edit("reset", func() { b.Reset() }),
		edit("type after reset", func() { typed(t, b, "x") }),
	}
	st := Styles{Atom: tcell.StyleDefault.Bold(true)}
	for _, s := range steps {
		before := b.View(st, true) // Lays it out, so a stale layout would be kept.
		text := b.Value()
		s.do()
		if s.edit && b.Value() == text {
			t.Fatalf("%s: changed nothing, so it proves nothing", s.name)
		}
		fresh := New(b.maxRows)
		fresh.SetWidth(b.width)
		fresh.SetAtoms(b.atom)
		fresh.SetValue(b.Value())
		fresh.SetPosition(b.Position())
		fresh.top = b.top
		fresh.layout = nil // Laid out from nothing kept, whatever its setup kept.
		got, want := b.laid(), fresh.laid()
		if !reflect.DeepEqual(got.rows, want.rows) || !reflect.DeepEqual(got.atoms, want.atoms) {
			t.Fatalf("%s: laid out %v %v, want %v %v", s.name, got.rows, got.atoms, want.rows, want.atoms)
		}
		if b.Rows() != fresh.Rows() || b.Height() != fresh.Height() {
			t.Fatalf("%s: %d rows, height %d; want %d, %d", s.name, b.Rows(), b.Height(), fresh.Rows(), fresh.Height())
		}
		gr, gc := b.Cursor()
		wr, wc := fresh.Cursor()
		if gr != wr || gc != wc {
			t.Fatalf("%s: cursor at %d,%d, want %d,%d", s.name, gr, gc, wr, wc)
		}
		view, wantView := b.View(st, true), fresh.View(st, true)
		if view != wantView {
			t.Fatalf("%s: view\n%q\nwant\n%q", s.name, view, wantView)
		}
		if view == before {
			t.Fatalf("%s: the view didn't change, so it proves nothing", s.name)
		}
	}
}
