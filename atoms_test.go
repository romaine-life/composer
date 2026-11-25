package composer

import (
	"regexp"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

var marked = regexp.MustCompile(`⟦[^⟦⟧\s]+⟧`)

func atomBox(width int, text string) *Box {
	b := boxOf(width, 5, text)
	b.SetAtoms(marked)
	return b
}

func TestBackspaceTakesAnAtomWhole(t *testing.T) {
	b := atomBox(40, "a ⟦tmux.window⟧")
	press(b, tcell.KeyBackspace2, 0)
	if b.Value() != "a " {
		t.Fatalf("value %q, want %q", b.Value(), "a ")
	}
	b.Undo()
	if b.Value() != "a ⟦tmux.window⟧" {
		t.Fatalf("undo: %q", b.Value())
	}
}

func TestDeleteTakesAnAtomWhole(t *testing.T) {
	b := atomBox(40, "⟦tmux.window⟧ b")
	b.SetPosition(0, 0)
	press(b, tcell.KeyDelete, 0)
	if b.Value() != " b" {
		t.Fatalf("value %q", b.Value())
	}
}

func TestTheCursorStepsOverAnAtom(t *testing.T) {
	b := atomBox(40, "a⟦tmux.window⟧b")
	b.SetPosition(0, 1)
	press(b, tcell.KeyRight, 0)
	if _, at := b.Position(); at != len("a⟦tmux.window⟧") {
		t.Fatalf("right: offset %d", at)
	}
	press(b, tcell.KeyLeft, 0)
	if _, at := b.Position(); at != 1 {
		t.Fatalf("left: offset %d", at)
	}
}

func TestTheCursorIsNeverPlacedInsideAnAtom(t *testing.T) {
	b := atomBox(40, "a ⟦tmux.window⟧")
	b.SetPosition(0, 6)
	if _, at := b.Position(); at != 2 {
		t.Fatalf("SetPosition: offset %d, want the atom's start", at)
	}
	b.Click(8, 0)
	if _, at := b.Position(); at != 2 {
		t.Fatalf("Click: offset %d, want the atom's start", at)
	}
	typed(t, b, "x")
	if b.Value() != "a x⟦tmux.window⟧" {
		t.Fatalf("typed at the atom: %q", b.Value())
	}
}

func TestAWordDeletionTakesAnAtomWhole(t *testing.T) {
	b := atomBox(40, "see ⟦at-claude.side-panel⟧")
	press(b, tcell.KeyCtrlW, 0)
	if b.Value() != "see " {
		t.Fatalf("Ctrl+W: %q", b.Value())
	}
	b = atomBox(40, "see ⟦at-claude.side-panel⟧ now")
	b.SetPosition(0, len("see "))
	press(b, tcell.KeyCtrlK, 0)
	if b.Value() != "see " {
		t.Fatalf("Ctrl+K: %q", b.Value())
	}
}

func TestAnAtomWrapsWhole(t *testing.T) {
	// Without atoms the hyphen is a place to break.
	b := atomBox(30, "abcdefghij ⟦at-claude.side-panel⟧")
	if b.Rows() != 2 {
		t.Fatalf("rows %d, want 2", b.Rows())
	}
	var second strings.Builder
	for _, c := range b.shown(Styles{})[1] {
		second.WriteString(c.text)
	}
	if second.String() != "⟦at-claude.side-panel⟧" {
		t.Fatalf("second row %q, want the atom whole", second.String())
	}
}

func TestAnAtomIsDrawnInItsStyle(t *testing.T) {
	b := atomBox(40, "a ⟦x.y⟧ b")
	st := Styles{Atom: tcell.StyleDefault.Reverse(true)}
	rows := b.shown(st)
	for _, c := range rows[0] {
		atom := c.off >= 2 && c.off < len("a ⟦x.y⟧")
		if (c.style == st.Atom) != atom {
			t.Fatalf("cell %q at %d drawn %v", c.text, c.off, c.style)
		}
	}
}

func TestPastedTextWithAnAtomKeepsIt(t *testing.T) {
	b := atomBox(40, "")
	b.Insert("look at ⟦tmux.pane⟧")
	press(b, tcell.KeyBackspace2, 0)
	if b.Value() != "look at " {
		t.Fatalf("value %q", b.Value())
	}
}

func TestWithoutAtomsNothingChanges(t *testing.T) {
	b := boxOf(40, 5, "a ⟦tmux.window⟧")
	press(b, tcell.KeyBackspace2, 0)
	if b.Value() != "a ⟦tmux.window" {
		t.Fatalf("value %q", b.Value())
	}
}

func TestReplaceTurnsAWordIntoAnAtom(t *testing.T) {
	b := atomBox(40, "the window is")
	b.SetPosition(0, len("the window"))
	b.Replace(0, len("the "), len("the window"), "⟦tmux.window⟧")
	if b.Value() != "the ⟦tmux.window⟧ is" {
		t.Fatalf("value %q", b.Value())
	}
	if _, at := b.Position(); at != len("the ⟦tmux.window⟧") {
		t.Fatalf("cursor at %d, want after the atom", at)
	}
	b.Undo()
	if b.Value() != "the window is" {
		t.Fatalf("one undo: %q", b.Value())
	}
}

func TestReplaceStartingInsideAnAtomTakesItWhole(t *testing.T) {
	b := atomBox(40, "x ⟦tmux.window⟧ z")
	b.Replace(0, len("x ⟦tmux"), len("x ⟦tmux.window⟧"), "y")
	if b.Value() != "x y z" {
		t.Fatalf("value %q, want %q", b.Value(), "x y z")
	}
	if _, at := b.Position(); at != len("x y") {
		t.Fatalf("cursor at %d, want %d", at, len("x y"))
	}
}
