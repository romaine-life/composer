package composer

import (
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

// rows lays the whole message out at the box's width.
func (b *Box) rows() []row {
	var all []row
	for i, p := range b.paras {
		for _, s := range wrap(p, b.textWidth(), b.atomsIn(p)) {
			all = append(all, row{i, s[0], s[1]})
		}
	}
	return all
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
		if _, _, in := within(atoms, off); !in && boundaries&uniseg.MaskLine == uniseg.LineCanBreak {
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
