package composer

import (
	"regexp"
	"sort"
	"unicode"
	"unicode/utf8"

	"github.com/rivo/uniseg"
)

// row is one screen row of the message: bytes start to end of a paragraph.
type row struct {
	para, start, end int
}

// textWidth is the columns text wraps to: one short of the box, so the cursor has
// a cell after a full row.
func (b *Box) textWidth() int {
	return max(1, b.width-1)
}

// layout is the message laid out, its rows and the cells they are drawn in: what
// depends only on its text, the box's width and what an atom is. It is kept until
// one of those changes, since an app asks for the box's height and cursor many
// times a frame and wrapping a long message is the slow part.
type layout struct {
	text  int            // The text it was laid out from, as setParas counts.
	width int            // The box's width it was laid out at.
	atom  *regexp.Regexp // The atom pattern it was laid out with.
	rows  []row
	atoms [][][]int // Each paragraph's atoms (atomsIn).

	cells [][]cell // Each row's cells (placed), filled as rows are first shown.
}

// setParas replaces the message's lines. Every change to the text goes through
// it, so the layout knows the text is new.
func (b *Box) setParas(paras []string) {
	b.paras = paras
	b.text++
}

// laid is the message laid out as it is now: the kept layout, or a new one when
// the text, the width or the atom pattern has changed since.
func (b *Box) laid() *layout {
	if l := b.layout; l != nil && l.text == b.text && l.width == b.width && l.atom == b.atom {
		return l
	}
	l := &layout{text: b.text, width: b.width, atom: b.atom, atoms: make([][][]int, len(b.paras))}
	for i, p := range b.paras {
		l.atoms[i] = b.atomsIn(p)
		for _, s := range wrap(p, b.textWidth(), l.atoms[i]) {
			l.rows = append(l.rows, row{i, s[0], s[1]})
		}
	}
	b.layout = l
	return l
}

// rows lays the whole message out at the box's width. The rows are shared with the
// kept layout, so they are only read.
func (b *Box) rows() []row {
	return b.laid().rows
}

// wrap breaks a paragraph into rows no wider than w columns, at the places Unicode
// allows a line break (after a word's trailing spaces), and inside a word only when
// the word alone is wider than a row. Spaces never start a row: they stay at the end
// of the one they follow, past its edge if they must, as a word processor keeps them.
// A grapheme (an emoji with its skin tone, a letter with its accents) is never split,
// nor an atom (atoms, its byte ranges in p) unless it alone is wider than a row.
func wrap(p string, w int, atoms [][]int) [][2]int {
	var rows [][2]int
	start, col := 0, 0
	brk, brkCol := -1, 0 // The last break opportunity in this row, and its column.
	state, rest, off := -1, p, 0
	a := 0 // The first atom not yet behind off: off only grows.
	for len(rest) > 0 {
		g, next, boundaries, st := uniseg.StepString(rest, state)
		gw := boundaries >> uniseg.ShiftWidth
		r, _ := utf8.DecodeRuneInString(g)
		for col+gw > w && !unicode.IsSpace(r) && off > start {
			if brk > start {
				rows = append(rows, [2]int{start, brk})
				start, col = brk, col-brkCol
			} else {
				rows = append(rows, [2]int{start, off})
				start, col = off, 0
			}
			brk = -1
		}
		col += gw
		off += len(g)
		for a < len(atoms) && atoms[a][1] <= off {
			a++
		}
		in := a < len(atoms) && atoms[a][0] < off
		if !in && boundaries&uniseg.MaskLine == uniseg.LineCanBreak {
			brk, brkCol = off, col
		}
		rest, state = next, st
	}
	return append(rows, [2]int{start, len(p)})
}

// graphemes is the offsets in s where a grapheme starts, and len(s).
func graphemes(s string) []int {
	offs := []int{0}
	state, rest, off := -1, s, 0
	for len(rest) > 0 {
		g, next, _, st := uniseg.StepString(rest, state)
		off += len(g)
		offs = append(offs, off)
		rest, state = next, st
	}
	return offs
}

// before is the grapheme boundary before offset i in s.
func before(s string, i int) int {
	offs := graphemes(s)
	k := sort.SearchInts(offs, i)
	return offs[max(0, k-1)]
}

// after is the grapheme boundary after offset i in s.
func after(s string, i int) int {
	offs := graphemes(s)
	k := sort.SearchInts(offs, i+1)
	return offs[min(len(offs)-1, k)]
}

// width is how many columns s takes.
func width(s string) int {
	return uniseg.StringWidth(s)
}

// cursorRow is the index of the row the cursor is on. An offset at the end of a
// wrapped row is the start of the next one.
func (b *Box) cursorRow(rows []row) int {
	for i, r := range rows {
		if r.para != b.line {
			continue
		}
		last := i+1 == len(rows) || rows[i+1].para != r.para
		if b.pos >= r.start && (b.pos < r.end || last) {
			return i
		}
	}
	return 0
}

// column is the cursor's column within its row, no further right than the cell
// kept for it.
func (b *Box) column(r row) int {
	return min(width(b.paras[r.para][r.start:b.pos]), b.textWidth())
}

// at is the offset in row r nearest column col without passing it. Only a
// paragraph's last row can put the cursor at its end; elsewhere that is the next
// row's start.
func (b *Box) at(rows []row, i, col int) int {
	r := rows[i]
	p := b.paras[r.para]
	last := i+1 == len(rows) || rows[i+1].para != r.para
	best := r.start
	c := 0
	state, rest, off := -1, p[r.start:r.end], r.start
	for len(rest) > 0 {
		g, next, boundaries, st := uniseg.StepString(rest, state)
		c += boundaries >> uniseg.ShiftWidth
		off += len(g)
		if c > col || (off == r.end && !last) {
			break
		}
		best = off
		rest, state = next, st
	}
	return b.snap(p, best)
}

// Rows is how many rows the whole message wraps to.
func (b *Box) Rows() int {
	return len(b.rows())
}

// Height is how many rows the box takes on screen: its rows, up to its maximum.
func (b *Box) Height() int {
	return min(b.Rows(), b.maxRows)
}

// Top is the row of the message shown first, 0 until the message outgrows the box
// and scrolls.
func (b *Box) Top() int {
	return b.top
}

// Cursor is where the cursor is within the box as shown: its row from the box's
// top and its column.
func (b *Box) Cursor() (row, col int) {
	rows := b.rows()
	i := b.cursorRow(rows)
	return i - b.top, b.column(rows[i])
}

// reveal scrolls so the cursor's row is shown, and no further than the last row.
func (b *Box) reveal() {
	rows := b.rows()
	h := min(len(rows), b.maxRows)
	i := b.cursorRow(rows)
	b.top = min(b.top, i)
	b.top = max(b.top, i-h+1)
	b.top = max(0, min(b.top, len(rows)-h))
}
