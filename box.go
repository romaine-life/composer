// Package composer is the message box shared by gvoice-tui and at-claude: the text,
// the cursor, wrapping and every editing key. It has no terminal of its own. An app
// passes it keys and pastes, asks what to show, and draws it where it wants: on a
// tcell screen with Draw, or as text for a Bubble Tea view with View.
package composer

import (
	"slices"
	"strings"
	"unicode"
)

// Result says what a key did.
type Result int

const (
	// Unhandled: the key isn't the box's, so the app may use it.
	Unhandled Result = iota
	// Handled: the box used the key, whether or not anything changed.
	Handled
	// Send: Enter. The box keeps the text; the app sends Value and resets it.
	Send
)

// undoLimit is how many changes Undo goes back through.
const undoLimit = 100

// tabSpaces is what a tab becomes, as Bubble Tea's textarea made it.
const tabSpaces = "    "

// Box is one message being written. Its zero value isn't usable; use New.
type Box struct {
	paras []string // The message's lines, which wrap to rows on screen.
	line  int      // The cursor's paragraph.
	pos   int      // The cursor's byte offset in it, always at a grapheme boundary.
	goal  int      // The column Up and Down aim for, or -1 to take the cursor's.

	width   int // Columns the box takes on screen.
	maxRows int // Rows it grows to before it scrolls.
	top     int // The first row shown.

	undo, redo []snapshot
	typing     bool // The last change was typed text, which more typing joins.
	lastTyped  rune
}

type snapshot struct {
	paras     []string
	line, pos int
}

// New returns an empty box that grows to maxRows rows and then scrolls.
func New(maxRows int) *Box {
	return &Box{paras: []string{""}, goal: -1, width: 2, maxRows: max(1, maxRows)}
}

// SetWidth sets the columns the box takes on screen. The text wraps one column
// short of it, so the cursor has room after a full row.
func (b *Box) SetWidth(w int) {
	if max(2, w) == b.width {
		return
	}
	b.width = max(2, w)
	b.goal = -1
	b.reveal()
}

// Width is the columns the box takes on screen.
func (b *Box) Width() int {
	return b.width
}

// SetMaxRows sets the rows the box grows to before it scrolls.
func (b *Box) SetMaxRows(n int) {
	b.maxRows = max(1, n)
	b.reveal()
}

// Value is the whole message, its lines joined by "\n".
func (b *Box) Value() string {
	return strings.Join(b.paras, "\n")
}

// Empty says nothing is typed.
func (b *Box) Empty() bool {
	return len(b.paras) == 1 && b.paras[0] == ""
}

// SetValue replaces the message, the cursor at its end, and starts a fresh undo
// history: a draft brought back or a sent message cleared isn't an edit to undo.
func (b *Box) SetValue(s string) {
	b.paras = strings.Split(clean(s), "\n")
	b.line = len(b.paras) - 1
	b.pos = len(b.paras[b.line])
	b.goal = -1
	b.undo, b.redo, b.typing = nil, nil, false
	b.top = 0
	b.reveal()
}

// Reset empties the box.
func (b *Box) Reset() {
	b.SetValue("")
}

// Insert puts text in at the cursor as one change, as a paste or a picked command
// does. Line breaks stay; a tab becomes spaces and other control characters are
// dropped, so nothing typed or pasted can reach the terminal as a command.
func (b *Box) Insert(s string) {
	s = clean(s)
	if s == "" {
		return
	}
	b.save(false, 0)
	b.insert(s)
}

// Paragraph is the cursor's line of the message and the cursor's byte offset in
// it, for an app that inserts with regard to the words around the cursor.
func (b *Box) Paragraph() (text string, at int) {
	return b.paras[b.line], b.pos
}

// Position is the cursor's line of the message and its byte offset in that line.
func (b *Box) Position() (line, offset int) {
	return b.line, b.pos
}

// SetPosition puts the cursor at a line of the message and a byte offset in it,
// each kept within the message and the offset moved back to a character's start.
func (b *Box) SetPosition(line, offset int) {
	b.line = min(max(0, line), len(b.paras)-1)
	p := b.paras[b.line]
	b.pos = min(max(0, offset), len(p))
	if b.pos < len(p) {
		b.pos = before(p, b.pos+1)
	}
	b.moved()
}

// MoveToEnd puts the cursor at the end of the message.
func (b *Box) MoveToEnd() {
	b.line = len(b.paras) - 1
	b.pos = len(b.paras[b.line])
	b.moved()
}

// clean makes text safe to keep and draw: "\r\n" and "\r" become "\n", a tab
// becomes spaces, and other control characters go.
func clean(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\t", tabSpaces)
	return strings.Map(func(r rune) rune {
		if r != '\n' && !kept(r) {
			return -1
		}
		return r
	}, s)
}

// kept says a character may be in a message: not a control character, nor one of
// the bidirectional controls that would reorder how the text around it is shown.
func kept(r rune) bool {
	return !unicode.IsControl(r) && !(r >= 0x202a && r <= 0x202e) && !(r >= 0x2066 && r <= 0x2069)
}

// insert puts clean text in at the cursor.
func (b *Box) insert(s string) {
	p := b.paras[b.line]
	before, after := p[:b.pos], p[b.pos:]
	lines := strings.Split(s, "\n")
	last := len(lines) - 1
	b.pos = len(lines[last])
	if last == 0 {
		b.pos += len(before)
	}
	lines[0] = before + lines[0]
	lines[last] += after
	b.paras = slices.Replace(b.paras, b.line, b.line+1, lines...)
	b.line += last
	b.changed()
}

// save records the message before a change, for Undo. Typed text joins the change
// before it, a word at a time: a typed space after a word starts a new one.
func (b *Box) save(typed bool, r rune) {
	join := typed && b.typing && !(unicode.IsSpace(r) && !unicode.IsSpace(b.lastTyped))
	b.typing, b.lastTyped = typed, r
	if join {
		return
	}
	b.undo = append(b.undo, b.snapshot())
	if len(b.undo) > undoLimit {
		b.undo = b.undo[len(b.undo)-undoLimit:]
	}
	b.redo = nil
}

func (b *Box) snapshot() snapshot {
	return snapshot{slices.Clone(b.paras), b.line, b.pos}
}

func (b *Box) restore(s snapshot) {
	b.paras, b.line, b.pos = s.paras, s.line, s.pos
	b.typing = false
	b.changed()
}

// Undo takes back the last change. It says whether there was one.
func (b *Box) Undo() bool {
	if len(b.undo) == 0 {
		return false
	}
	b.redo = append(b.redo, b.snapshot())
	s := b.undo[len(b.undo)-1]
	b.undo = b.undo[:len(b.undo)-1]
	b.restore(s)
	return true
}

// Redo puts back what Undo took. It says whether there was anything.
func (b *Box) Redo() bool {
	if len(b.redo) == 0 {
		return false
	}
	b.undo = append(b.undo, b.snapshot())
	s := b.redo[len(b.redo)-1]
	b.redo = b.redo[:len(b.redo)-1]
	b.restore(s)
	return true
}

// changed follows an edit: the column Up and Down aim for is the cursor's again,
// and the cursor stays in view.
func (b *Box) changed() {
	b.goal = -1
	b.reveal()
}

// moved follows a cursor move that isn't Up or Down; it ends a run of typing.
func (b *Box) moved() {
	b.typing = false
	b.goal = -1
	b.reveal()
}

// remove deletes from paragraph l1 offset p1 to paragraph l2 offset p2, the cursor
// left where the deletion started.
func (b *Box) remove(l1, p1, l2, p2 int) {
	if l1 == l2 && p1 == p2 {
		return
	}
	joined := b.paras[l1][:p1] + b.paras[l2][p2:]
	b.paras = slices.Replace(b.paras, l1, l2+1, joined)
	b.line, b.pos = l1, p1
	b.changed()
}
