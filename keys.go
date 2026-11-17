package composer

import (
	"unicode"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
)

// Key takes a key as tcell reports it. Enter sends. Shift+Enter, Alt+Enter and
// Ctrl+J start a new line; Windows Terminal, as at-terminal sets it up, sends
// Shift+Enter as Alt+Enter. Keys the box doesn't use (Esc, Tab, PgUp, Ctrl+C, Up
// on the first row, and so on) come back Unhandled for the app.
func (b *Box) Key(k tcell.Key, r rune, mod tcell.ModMask) Result {
	word := mod&(tcell.ModCtrl|tcell.ModAlt) != 0
	switch k {
	case tcell.KeyRune:
		return b.runeKey(r, mod)
	case tcell.KeyEnter:
		if mod&(tcell.ModShift|tcell.ModAlt) != 0 {
			return b.do(NewLine)
		}
		return Send
	case tcell.KeyCtrlJ:
		return b.do(NewLine)
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if mod&tcell.ModAlt != 0 || (k == tcell.KeyBackspace2 && mod&tcell.ModCtrl != 0) {
			b.deleteWordBack()
		} else {
			b.deleteBack()
		}
	case tcell.KeyCtrlW:
		b.deleteWordBack()
	case tcell.KeyDelete:
		if word {
			b.deleteWordForward()
		} else {
			b.deleteForward()
		}
	case tcell.KeyCtrlD:
		b.deleteForward()
	case tcell.KeyCtrlK:
		b.deleteToEnd()
	case tcell.KeyCtrlU:
		b.deleteToStart()
	case tcell.KeyLeft:
		if word {
			b.wordLeft()
		} else {
			b.left()
		}
	case tcell.KeyRight:
		if word {
			b.wordRight()
		} else {
			b.right()
		}
	case tcell.KeyUp:
		if mod&tcell.ModCtrl != 0 {
			return b.do(PreviousParagraph)
		}
		return b.vertical(-1)
	case tcell.KeyDown:
		if mod&tcell.ModCtrl != 0 {
			return b.do(NextParagraph)
		}
		return b.vertical(1)
	case tcell.KeyHome, tcell.KeyCtrlA:
		if k == tcell.KeyHome && mod&tcell.ModCtrl != 0 {
			return Unhandled
		}
		b.rowStart()
	case tcell.KeyEnd, tcell.KeyCtrlE:
		if k == tcell.KeyEnd && mod&tcell.ModCtrl != 0 {
			return Unhandled
		}
		b.rowEnd()
	case tcell.KeyCtrlZ:
		return b.do(Undo)
	case tcell.KeyCtrlY:
		return b.do(Redo)
	default:
		return Unhandled
	}
	return Handled
}

// runeKey types a character, or with Alt runs the word keys a terminal sends for
// Alt+B, Alt+F and Alt+D.
func (b *Box) runeKey(r rune, mod tcell.ModMask) Result {
	if mod&tcell.ModAlt != 0 {
		switch r {
		case 'b':
			b.wordLeft()
		case 'f':
			b.wordRight()
		case 'd':
			b.deleteWordForward()
		default:
			return Unhandled
		}
		return Handled
	}
	if mod&tcell.ModCtrl != 0 || !kept(r) {
		return Unhandled
	}
	b.save(true, r)
	b.insert(string(r))
	return Handled
}

func (b *Box) left() {
	if b.pos > 0 {
		b.pos = before(b.paras[b.line], b.pos)
	} else if b.line > 0 {
		b.line--
		b.pos = len(b.paras[b.line])
	}
	b.moved()
}

func (b *Box) right() {
	if b.pos < len(b.paras[b.line]) {
		b.pos = after(b.paras[b.line], b.pos)
	} else if b.line < len(b.paras)-1 {
		b.line++
		b.pos = 0
	}
	b.moved()
}

// wordStart is the start of the word at or before offset i in s, past any spaces
// before i.
func wordStart(s string, i int) int {
	for i > 0 {
		r, _ := utf8.DecodeLastRuneInString(s[:i])
		if !unicode.IsSpace(r) {
			break
		}
		i = before(s, i)
	}
	for i > 0 {
		r, _ := utf8.DecodeLastRuneInString(s[:i])
		if unicode.IsSpace(r) {
			break
		}
		i = before(s, i)
	}
	return i
}

// wordEnd is the end of the word at or after offset i in s, past any spaces after i.
func wordEnd(s string, i int) int {
	for i < len(s) {
		r, _ := utf8.DecodeRuneInString(s[i:])
		if !unicode.IsSpace(r) {
			break
		}
		i = after(s, i)
	}
	for i < len(s) {
		r, _ := utf8.DecodeRuneInString(s[i:])
		if unicode.IsSpace(r) {
			break
		}
		i = after(s, i)
	}
	return i
}

func (b *Box) wordLeft() {
	if b.pos == 0 {
		b.left()
		return
	}
	b.pos = wordStart(b.paras[b.line], b.pos)
	b.moved()
}

func (b *Box) wordRight() {
	if b.pos == len(b.paras[b.line]) {
		b.right()
		return
	}
	b.pos = wordEnd(b.paras[b.line], b.pos)
	b.moved()
}

func (b *Box) deleteBack() {
	switch {
	case b.pos > 0:
		b.save(false, 0)
		b.remove(b.line, before(b.paras[b.line], b.pos), b.line, b.pos)
	case b.line > 0:
		b.save(false, 0)
		b.remove(b.line-1, len(b.paras[b.line-1]), b.line, 0)
	}
}

func (b *Box) deleteForward() {
	switch {
	case b.pos < len(b.paras[b.line]):
		b.save(false, 0)
		b.remove(b.line, b.pos, b.line, after(b.paras[b.line], b.pos))
	case b.line < len(b.paras)-1:
		b.save(false, 0)
		b.remove(b.line, b.pos, b.line+1, 0)
	}
}

func (b *Box) deleteWordBack() {
	if b.pos == 0 {
		b.deleteBack()
		return
	}
	b.save(false, 0)
	b.remove(b.line, wordStart(b.paras[b.line], b.pos), b.line, b.pos)
}

func (b *Box) deleteWordForward() {
	if b.pos == len(b.paras[b.line]) {
		b.deleteForward()
		return
	}
	b.save(false, 0)
	b.remove(b.line, b.pos, b.line, wordEnd(b.paras[b.line], b.pos))
}

// deleteToEnd deletes to the end of the cursor's line of the message, or at its end
// joins the next line to it.
func (b *Box) deleteToEnd() {
	if b.pos == len(b.paras[b.line]) {
		b.deleteForward()
		return
	}
	b.save(false, 0)
	b.remove(b.line, b.pos, b.line, len(b.paras[b.line]))
}

// deleteToStart deletes from the start of the cursor's line of the message to it.
func (b *Box) deleteToStart() {
	if b.pos == 0 {
		return
	}
	b.save(false, 0)
	b.remove(b.line, 0, b.line, b.pos)
}

// vertical moves the cursor a row up or down, aiming for the column it started
// from however short the rows between. Past the first or last row it is Unhandled,
// for the app to scroll what is above the box.
func (b *Box) vertical(d int) Result {
	rows := b.rows()
	i := b.cursorRow(rows)
	if i+d < 0 || i+d >= len(rows) {
		return Unhandled
	}
	if b.goal < 0 {
		b.goal = b.column(rows[i])
	}
	goal := b.goal
	b.line = rows[i+d].para
	b.pos = b.at(rows, i+d, goal)
	b.typing = false
	b.reveal()
	b.goal = goal
	return Handled
}

func (b *Box) rowStart() {
	rows := b.rows()
	b.pos = rows[b.cursorRow(rows)].start
	b.moved()
}

// rowEnd moves to the end of the cursor's row; on a wrapped row, before its last
// character, since its end is where the next row starts.
func (b *Box) rowEnd() {
	rows := b.rows()
	i := b.cursorRow(rows)
	b.pos = b.at(rows, i, b.width)
	if i+1 == len(rows) || rows[i+1].para != rows[i].para {
		b.pos = rows[i].end
	}
	b.moved()
}

func (b *Box) nextParagraph() {
	if b.line < len(b.paras)-1 {
		b.line++
		b.pos = 0
	} else {
		b.pos = len(b.paras[b.line])
	}
	b.moved()
}

func (b *Box) previousParagraph() {
	if b.pos == 0 && b.line > 0 {
		b.line--
	}
	b.pos = 0
	b.moved()
}
