package composer

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func typed(t *testing.T, b *Box, s string) {
	t.Helper()
	for _, r := range s {
		if got := b.Key(tcell.KeyRune, r, 0); got != Handled {
			t.Fatalf("typing %q: %v", r, got)
		}
	}
}

func press(b *Box, k tcell.Key, mod tcell.ModMask) Result {
	return b.Key(k, 0, mod)
}

func boxOf(width, rows int, text string) *Box {
	b := New(rows)
	b.SetWidth(width)
	b.SetValue(text)
	return b
}

func TestEnterSendsAndKeepsTheText(t *testing.T) {
	b := New(5)
	typed(t, b, "hi")
	if got := press(b, tcell.KeyEnter, 0); got != Send {
		t.Fatalf("Enter: %v, want Send", got)
	}
	if b.Value() != "hi" {
		t.Fatalf("value %q", b.Value())
	}
}

func TestNewLineKeys(t *testing.T) {
	for name, key := range map[string]struct {
		k   tcell.Key
		mod tcell.ModMask
	}{
		"Shift+Enter": {tcell.KeyEnter, tcell.ModShift},
		"Alt+Enter":   {tcell.KeyEnter, tcell.ModAlt},
		"Ctrl+J":      {tcell.KeyCtrlJ, tcell.ModCtrl},
	} {
		b := New(5)
		typed(t, b, "a")
		if got := press(b, key.k, key.mod); got != Handled {
			t.Fatalf("%s: %v", name, got)
		}
		typed(t, b, "b")
		if b.Value() != "a\nb" {
			t.Errorf("%s: value %q", name, b.Value())
		}
	}
}

func TestInsertKeepsLineBreaksAndDropsControls(t *testing.T) {
	b := New(5)
	b.Insert("one\r\ntwo\rthree\tfour\x1b[31m\x07‮")
	if want := "one\ntwo\nthree    four[31m"; b.Value() != want {
		t.Fatalf("value %q, want %q", b.Value(), want)
	}
	b.Undo()
	if !b.Empty() {
		t.Fatalf("a paste is one change; after undo %q", b.Value())
	}
}

func TestInsertGoesInAtTheCursor(t *testing.T) {
	b := boxOf(40, 5, "ad")
	press(b, tcell.KeyLeft, 0)
	b.Insert("b\nc")
	if b.Value() != "ab\ncd" {
		t.Fatalf("value %q", b.Value())
	}
	if text, at := b.Paragraph(); text != "cd" || at != 1 {
		t.Fatalf("paragraph %q at %d", text, at)
	}
}

func shownRows(b *Box) []string {
	var out []string
	for _, r := range b.rows() {
		out = append(out, b.paras[r.para][r.start:r.end])
	}
	return out
}

func TestWrapsAtWordsAndSplitsOnlyLongWords(t *testing.T) {
	// Width 8 leaves 7 columns for text and one for the cursor.
	b := boxOf(8, 10, "the quick brown fox\nabcdefghijklmnop")
	want := []string{"the ", "quick ", "brown ", "fox", "abcdefg", "hijklmn", "op"}
	if got := shownRows(b); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("rows %q, want %q", got, want)
	}
}

func TestSpacesStayAtTheEndOfTheirRow(t *testing.T) {
	b := boxOf(6, 10, "abcde     fg")
	if got := shownRows(b); strings.Join(got, "|") != "abcde     |fg" {
		t.Fatalf("rows %q", got)
	}
}

func TestGraphemesAreNeverSplit(t *testing.T) {
	// A thumbs-up with a skin tone is two code points, one character, two columns.
	b := boxOf(4, 10, "a👍🏽b")
	if got := shownRows(b); strings.Join(got, "|") != "a👍🏽|b" {
		t.Fatalf("rows %q", got)
	}
	press(b, tcell.KeyLeft, 0)
	press(b, tcell.KeyLeft, 0)
	if _, at := b.Paragraph(); at != 1 {
		t.Fatalf("Left twice from the end lands at %d, want 1 (before the emoji)", at)
	}
	press(b, tcell.KeyDelete, 0)
	if b.Value() != "ab" {
		t.Fatalf("Delete took part of the emoji: %q", b.Value())
	}
}

func TestUpAndDownKeepTheirColumn(t *testing.T) {
	b := boxOf(40, 10, "abcdefgh\nab\nabcdefgh")
	if got := press(b, tcell.KeyUp, 0); got != Handled {
		t.Fatalf("Up: %v", got)
	}
	if text, at := b.Paragraph(); text != "ab" || at != 2 {
		t.Fatalf("after Up: %q at %d", text, at)
	}
	press(b, tcell.KeyUp, 0)
	if text, at := b.Paragraph(); text != "abcdefgh" || at != 8 {
		t.Fatalf("after Up again the column comes back: %q at %d", text, at)
	}
	if got := press(b, tcell.KeyUp, 0); got != Unhandled {
		t.Fatalf("Up on the first row: %v, want Unhandled", got)
	}
	press(b, tcell.KeyEnd, tcell.ModNone)
	press(b, tcell.KeyDown, 0)
	press(b, tcell.KeyDown, 0)
	if got := press(b, tcell.KeyDown, 0); got != Unhandled {
		t.Fatalf("Down on the last row: %v, want Unhandled", got)
	}
}

func TestUpAndDownMoveBetweenWrappedRows(t *testing.T) {
	b := boxOf(8, 10, "the quick brown")
	press(b, tcell.KeyUp, 0)
	if row, col := b.Cursor(); row != 1 || col != 5 {
		t.Fatalf("cursor %d,%d, want 1,5", row, col)
	}
}

func TestParagraphKeys(t *testing.T) {
	b := boxOf(40, 10, "one\ntwo\nthree")
	press(b, tcell.KeyUp, tcell.ModCtrl)
	if text, at := b.Paragraph(); text != "three" || at != 0 {
		t.Fatalf("Ctrl+Up: %q at %d", text, at)
	}
	press(b, tcell.KeyUp, tcell.ModCtrl)
	if text, at := b.Paragraph(); text != "two" || at != 0 {
		t.Fatalf("Ctrl+Up again: %q at %d", text, at)
	}
	press(b, tcell.KeyDown, tcell.ModCtrl)
	press(b, tcell.KeyDown, tcell.ModCtrl)
	if text, at := b.Paragraph(); text != "three" || at != 5 {
		t.Fatalf("Ctrl+Down from the last line goes to its end: %q at %d", text, at)
	}
}

func TestDeletingWordsAndLines(t *testing.T) {
	b := boxOf(40, 10, "one two  ")
	press(b, tcell.KeyCtrlW, tcell.ModCtrl)
	if b.Value() != "one " {
		t.Fatalf("Ctrl+W: %q", b.Value())
	}
	press(b, tcell.KeyBackspace2, tcell.ModAlt)
	if b.Value() != "" {
		t.Fatalf("Alt+Backspace: %q", b.Value())
	}
	b.SetValue("ab\ncd")
	press(b, tcell.KeyCtrlU, tcell.ModCtrl)
	press(b, tcell.KeyBackspace2, 0)
	if b.Value() != "ab" {
		t.Fatalf("Backspace at a line's start joins it to the one before: %q", b.Value())
	}
	press(b, tcell.KeyHome, 0)
	press(b, tcell.KeyCtrlK, tcell.ModCtrl)
	if !b.Empty() {
		t.Fatalf("Ctrl+K: %q", b.Value())
	}
}

func TestShiftBackspaceDeletes(t *testing.T) {
	b := boxOf(40, 10, "Ab")
	press(b, tcell.KeyBackspace2, tcell.ModShift)
	if b.Value() != "A" {
		t.Fatalf("Shift+Backspace: %q", b.Value())
	}
}

func TestUndoGoesBackAWordAtATime(t *testing.T) {
	b := New(5)
	typed(t, b, "hello world")
	b.Undo()
	if b.Value() != "hello" {
		t.Fatalf("first undo: %q", b.Value())
	}
	press(b, tcell.KeyCtrlZ, tcell.ModCtrl)
	if !b.Empty() {
		t.Fatalf("second undo: %q", b.Value())
	}
	press(b, tcell.KeyCtrlY, tcell.ModCtrl)
	if b.Value() != "hello" {
		t.Fatalf("redo: %q", b.Value())
	}
	typed(t, b, "!")
	if b.Redo() {
		t.Fatal("an edit after undo leaves nothing to redo")
	}
}

func TestSetValueIsNotAnEdit(t *testing.T) {
	b := New(5)
	typed(t, b, "draft")
	b.SetValue("other")
	if b.Undo() {
		t.Fatal("undo went back past a draft brought back")
	}
}

func TestGrowsToMaxRowsThenScrolls(t *testing.T) {
	b := boxOf(20, 2, "1\n2\n3\n4")
	if b.Rows() != 4 || b.Height() != 2 {
		t.Fatalf("rows %d height %d", b.Rows(), b.Height())
	}
	if row, _ := b.Cursor(); row != 1 {
		t.Fatalf("cursor row %d, want the last shown", row)
	}
	press(b, tcell.KeyUp, 0)
	press(b, tcell.KeyUp, 0)
	if row, _ := b.Cursor(); row != 0 {
		t.Fatalf("cursor row %d after scrolling up", row)
	}
	if v := b.View(Styles{}, false); !strings.Contains(v, "2") || strings.Contains(v, "4") {
		t.Fatalf("view shows the wrong rows: %q", v)
	}
	if New(3).Height() != 1 {
		t.Fatal("an empty box takes one row")
	}
}

func TestKeysTheBoxLeavesToTheApp(t *testing.T) {
	b := boxOf(20, 5, "text")
	for name, k := range map[string]struct {
		k   tcell.Key
		r   rune
		mod tcell.ModMask
	}{
		"Esc": {tcell.KeyEscape, 0, 0}, "Tab": {tcell.KeyTab, 0, 0}, "PgUp": {tcell.KeyPgUp, 0, 0},
		"Ctrl+C": {tcell.KeyCtrlC, 0, tcell.ModCtrl}, "Ctrl+Home": {tcell.KeyHome, 0, tcell.ModCtrl},
		"Ctrl+End": {tcell.KeyEnd, 0, tcell.ModCtrl}, "Alt+x": {tcell.KeyRune, 'x', tcell.ModAlt},
	} {
		if got := b.Key(k.k, k.r, k.mod); got != Unhandled {
			t.Errorf("%s: %v, want Unhandled", name, got)
		}
	}
	if b.Value() != "text" {
		t.Fatalf("value changed: %q", b.Value())
	}
}

func TestViewPadsRowsAndDrawsTheCursor(t *testing.T) {
	st := Styles{Cursor: tcell.StyleDefault.Reverse(true), Placeholder: tcell.StyleDefault.Dim(true), PlaceholderText: "Message"}
	b := boxOf(10, 3, "")
	v := b.View(st, true)
	if !strings.Contains(v, "\x1b[0;7mM\x1b[0;2me") {
		t.Fatalf("placeholder with the cursor on its first letter: %q", v)
	}
	b.SetValue("hi")
	v = b.View(st, true)
	if !strings.HasPrefix(v, "\x1b[0mh\x1b[0mi\x1b[0m\x1b[0;7m \x1b[0m       \x1b[0m") {
		t.Fatalf("view %q", v)
	}
	if strings.Count(stripped(v), " ") != 8 {
		t.Fatalf("row not padded to the width: %q", stripped(v))
	}
}

func TestViewUsesColors(t *testing.T) {
	b := boxOf(10, 3, "x")
	v := b.View(Styles{Text: tcell.StyleDefault.Foreground(tcell.PaletteColor(240)).Background(tcell.ColorNavy)}, false)
	if !strings.Contains(v, "\x1b[0;38;5;240;44mx") {
		t.Fatalf("view %q", v)
	}
}

func stripped(s string) string {
	var out strings.Builder
	for len(s) > 0 {
		if s[0] == '\x1b' {
			s = s[strings.IndexByte(s, 'm')+1:]
			continue
		}
		out.WriteByte(s[0])
		s = s[1:]
	}
	return out.String()
}

type fakeScreen struct {
	cells            map[[2]int]string
	cursorX, cursorY int
}

func (f *fakeScreen) SetContent(x, y int, primary rune, combining []rune, _ tcell.Style) {
	f.cells[[2]int{x, y}] = string(append([]rune{primary}, combining...))
}

func (f *fakeScreen) ShowCursor(x, y int) {
	f.cursorX, f.cursorY = x, y
}

func TestDrawPlacesTextAndTheCursor(t *testing.T) {
	b := boxOf(5, 3, "ab👍🏽")
	s := &fakeScreen{cells: map[[2]int]string{}}
	b.Draw(s, 2, 7, Styles{}, true)
	if s.cells[[2]int{2, 7}] != "a" || s.cells[[2]int{4, 7}] != "👍🏽" || s.cells[[2]int{6, 7}] != " " {
		t.Fatalf("cells %v", s.cells)
	}
	if s.cursorX != 6 || s.cursorY != 7 {
		t.Fatalf("cursor at %d,%d, want 6,7", s.cursorX, s.cursorY)
	}
}

func TestClickPlacesTheCursor(t *testing.T) {
	b := boxOf(20, 5, "one\ntwo words")
	b.Click(5, 1)
	if text, at := b.Paragraph(); text != "two words" || at != 5 {
		t.Fatalf("click: %q at %d", text, at)
	}
	b.Click(50, 0)
	if text, at := b.Paragraph(); text != "one" || at != 3 {
		t.Fatalf("click past the end: %q at %d", text, at)
	}
}

func TestCommandsRunByName(t *testing.T) {
	b := boxOf(20, 5, "a")
	for _, c := range Commands {
		if !b.Do(c.Name) {
			t.Errorf("%s isn't run by Do", c.Name)
		}
		if c.Description == "" || c.Keys == "" {
			t.Errorf("%s lacks a description or keys", c.Name)
		}
	}
	if b.Do("nothing") {
		t.Error("Do ran an unknown command")
	}
}

func TestPositionAndTop(t *testing.T) {
	b := boxOf(20, 2, "a👍🏽b\nc\nd")
	if line, off := b.Position(); line != 2 || off != 1 || b.Top() != 1 {
		t.Fatalf("position %d,%d top %d", line, off, b.Top())
	}
	b.SetPosition(0, 3) // Inside the emoji: back to its start.
	if line, off := b.Position(); line != 0 || off != 1 || b.Top() != 0 {
		t.Fatalf("position %d,%d top %d", line, off, b.Top())
	}
	b.SetPosition(9, 9)
	if line, off := b.Position(); line != 2 || off != 1 {
		t.Fatalf("position %d,%d", line, off)
	}
}

func TestSettingTheSameWidthKeepsTheColumnUpAndDownAimFor(t *testing.T) {
	b := boxOf(40, 10, "abcdefgh\nab\nabcdefgh")
	press(b, tcell.KeyUp, 0)
	b.SetWidth(40)
	press(b, tcell.KeyUp, 0)
	if _, at := b.Paragraph(); at != 8 {
		t.Fatalf("column lost: %d", at)
	}
}
