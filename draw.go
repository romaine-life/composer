package composer

import (
	"fmt"
	"slices"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/uniseg"
)

// Styles are how the box draws: the message, the placeholder shown while it is
// empty, and, in View, the cell under the cursor.
type Styles struct {
	Text        tcell.Style
	Placeholder tcell.Style
	Cursor      tcell.Style
	// Atom is how an atom (SetAtoms) is drawn; left unset, as the message is.
	Atom tcell.Style
	// Placeholder text, shown while nothing is typed; "" shows none.
	PlaceholderText string
}

// Screen is what Draw draws on; a tcell.Screen is one.
type Screen interface {
	SetContent(x, y int, primary rune, combining []rune, style tcell.Style)
	ShowCursor(x, y int)
}

// cell is one grapheme placed in a shown row.
type cell struct {
	text  string
	col   int
	width int
	style tcell.Style
	off   int  // Its byte offset in the text it was placed from.
	atom  bool // It is part of an atom.
}

// shown lays out the rows on screen, each its graphemes within the box's width.
func (b *Box) shown(st Styles) [][]cell {
	h := b.Height()
	out := make([][]cell, h)
	if b.Empty() {
		out[0] = cells(st.PlaceholderText, b.width, st.Placeholder)
		return out
	}
	atom := st.Atom
	if atom == tcell.StyleDefault {
		atom = st.Text
	}
	l := b.laid()
	for i := range h {
		out[i] = slices.Clone(b.placed(l, b.top+i))
		for k, c := range out[i] {
			out[i][k].style = st.Text
			if c.atom {
				out[i][k].style = atom
			}
		}
	}
	return out
}

// placed is row i of layout l as cells, unstyled, placed once and kept with the
// layout. An empty row has none, and costs nothing to place again.
func (b *Box) placed(l *layout, i int) []cell {
	if l.cells == nil {
		l.cells = make([][]cell, len(l.rows))
	}
	if l.cells[i] == nil {
		r := l.rows[i]
		cs := cells(b.paras[r.para][r.start:r.end], b.width, tcell.StyleDefault)
		for k, c := range cs {
			cs[k].atom = in(l.atoms[r.para], r.start+c.off)
		}
		l.cells[i] = cs
	}
	return l.cells[i]
}

// cells places s's graphemes from column 0, up to w columns.
func cells(s string, w int, style tcell.Style) []cell {
	var out []cell
	col, off := 0, 0
	state := -1
	for len(s) > 0 {
		g, rest, boundaries, st := uniseg.StepString(s, state)
		gw := boundaries >> uniseg.ShiftWidth
		s, state = rest, st
		at := off
		off += len(g)
		if gw == 0 {
			continue
		}
		if col+gw > w {
			break
		}
		out = append(out, cell{text: g, col: col, width: gw, style: style, off: at})
		col += gw
	}
	return out
}

// Draw draws the box with its top left at x, y, Width columns wide and Height rows
// tall, and with focused puts the terminal's cursor where the box's is.
func (b *Box) Draw(s Screen, x, y int, st Styles, focused bool) {
	for i, cs := range b.shown(st) {
		for c := range b.width {
			s.SetContent(x+c, y+i, ' ', nil, st.Text)
		}
		for _, c := range cs {
			r := []rune(c.text)
			s.SetContent(x+c.col, y+i, r[0], r[1:], c.style)
		}
	}
	if focused {
		row, col := b.Cursor()
		s.ShowCursor(x+col, y+row)
	}
}

// View is the box as text for a view built of strings (Bubble Tea's): Height rows,
// each Width columns, styled with ANSI escapes, and with focused the cell under
// the cursor drawn in Styles.Cursor.
func (b *Box) View(st Styles, focused bool) string {
	crow, ccol := -1, -1
	if focused {
		crow, ccol = b.Cursor()
	}
	var out strings.Builder
	escapes := map[tcell.Style]string{} // Each style's escape, made once.
	escape := func(style tcell.Style) string {
		e, ok := escapes[style]
		if !ok {
			e = sgr(style)
			escapes[style] = e
		}
		return e
	}
	for i, cs := range b.shown(st) {
		if i > 0 {
			out.WriteByte('\n')
		}
		col := 0
		for _, c := range cs {
			style := c.style
			if i == crow && c.col == ccol {
				style = st.Cursor
			}
			out.WriteString(escape(style) + c.text)
			col = c.col + c.width
		}
		if i == crow && ccol >= col {
			out.WriteString(escape(st.Text) + strings.Repeat(" ", ccol-col) + escape(st.Cursor) + " ")
			col = ccol + 1
		}
		out.WriteString(escape(st.Text) + strings.Repeat(" ", max(0, b.width-col)) + "\x1b[0m")
	}
	return out.String()
}

// sgr is the ANSI escape that starts drawing in style.
func sgr(style tcell.Style) string {
	fg, bg, attrs := style.Decompose()
	codes := []string{"0"}
	for _, a := range []struct {
		mask tcell.AttrMask
		code string
	}{
		{tcell.AttrBold, "1"}, {tcell.AttrDim, "2"}, {tcell.AttrItalic, "3"},
		{tcell.AttrUnderline, "4"}, {tcell.AttrBlink, "5"}, {tcell.AttrReverse, "7"},
		{tcell.AttrStrikeThrough, "9"},
	} {
		if attrs&a.mask != 0 {
			codes = append(codes, a.code)
		}
	}
	codes = append(codes, color(fg, 30)...)
	codes = append(codes, color(bg, 40)...)
	return "\x1b[" + strings.Join(codes, ";") + "m"
}

// color is the SGR codes for a color, base 30 for the foreground and 40 for the
// background; none for the terminal's default.
func color(c tcell.Color, base int) []string {
	switch {
	case c == tcell.ColorDefault || !c.Valid():
		return nil
	case c.IsRGB():
		r, g, bl := c.RGB()
		return []string{fmt.Sprint(base + 8), "2", fmt.Sprint(r), fmt.Sprint(g), fmt.Sprint(bl)}
	}
	n := int(c - tcell.ColorValid)
	switch {
	case n < 8:
		return []string{fmt.Sprint(base + n)}
	case n < 16:
		return []string{fmt.Sprint(base + 60 + n - 8)}
	}
	return []string{fmt.Sprint(base + 8), "5", fmt.Sprint(n)}
}

// Click puts the cursor at a cell of the box as shown, x and y from its top left.
func (b *Box) Click(x, y int) {
	rows := b.rows()
	i := min(max(0, b.top+y), len(rows)-1)
	b.line = rows[i].para
	b.pos = b.at(rows, i, max(0, x))
	b.moved()
}
