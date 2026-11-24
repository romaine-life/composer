# composer

The message box shared by Nelson's terminal apps, gvoice-tui and at-claude: one place
for the text, the cursor, wrapping and every editing key, so a change to how typing
works reaches both. Work package:
[APPS-67](https://openproject.home.romaine.life/wp/APPS-67).

It is a Go library with no terminal of its own, like fzt: an app passes it keys and
pastes, and draws it where it wants.

## Keys

| Key | Does |
|---|---|
| Enter | Returns `Send`; the app sends `Value()` and resets the box |
| Shift+Enter, Alt+Enter, Ctrl+J | New line (at-terminal sends Shift+Enter as Alt+Enter) |
| Left, Right; Ctrl or Alt with them, Alt+B, Alt+F | A character; a word |
| Up, Down | A row, keeping the column; past the first or last row, `Unhandled` |
| Ctrl+Up, Ctrl+Down | The start of this line of the message or the one before; the next |
| Home, End, Ctrl+A, Ctrl+E | The start or end of the row |
| Backspace, Delete, Ctrl+H, Ctrl+D (Shift held too) | A character |
| Ctrl+W, Alt+Backspace, Ctrl+Backspace; Alt+D, Ctrl+Delete | A word back; forward |
| Ctrl+U, Ctrl+K | To the start, or the end, of this line of the message |
| Ctrl+Z, Ctrl+Y | Undo, a typed word at a time; redo |

Every other key comes back `Unhandled` for the app: Esc, Tab, PgUp and PgDn,
Ctrl+Home and Ctrl+End, Ctrl+C, other Alt keys. An app handles its own keys before
giving the rest to the box. `Commands` lists the box's commands with their keys, for
an app's command palette, and `Do` runs one.

## Using it

```go
box := composer.New(6)   // grows to 6 rows, then scrolls
box.SetWidth(w)          // columns it takes; text wraps one short, for the cursor
switch box.Key(ev.Key(), ev.Rune(), ev.Modifiers()) {
case composer.Send:
	send(box.Value())
case composer.Unhandled:
	// the app's own key
}
box.Insert(pasted)       // a paste, or text a command puts in, as one change
box.Draw(screen, x, y, styles, true)   // on a tcell screen, its cursor shown
view := box.View(styles, true)         // or as text for a Bubble Tea view
```

`Height` is the rows it takes, `Top` the message's first row shown, `Cursor` where
its cursor is within them, `Click` places the cursor at a cell, `Position` and
`SetPosition` give and set it as a line of the message and an offset, `Paragraph` is
the cursor's line, `Replace` puts text in place of part of a line as one change, and
`SetValue` and `Value` keep a draft.

`SetAtoms(pattern)` makes each stretch of text the pattern matches an atom, one unit:
the cursor steps over it and never stops inside it, Backspace and Delete take it
whole, a deletion that would cut into it takes all of it, it wraps whole unless it is
wider than a row, and it is drawn in `Styles.Atom`. Atoms are the message's text, so
`Value`, `SetValue`, a paste and undo carry them with nothing else to keep. at-claude
uses them for the objects picked from its menu
([ATC-191](https://openproject.home.romaine.life/wp/ATC-191)).

Text wraps at the places Unicode allows a line break, a word wider than a row is
split, spaces stay at the end of the row they follow, and a character made of
several code points (an emoji with its skin tone) is never split. What is typed or
pasted keeps its line breaks; a tab becomes four spaces, and other control characters
and the bidirectional controls that reorder text are dropped, so nothing in it
reaches the terminal as a command or changes how the text around it shows.

## Testing

```
go vet ./...
go test -p 4 ./...
```

## Shipping

Forgejo is its home; GitHub gets a one-way copy so `go` can fetch it by its
`github.com/romaine-life/composer` name. CI is pipeline-templates' go and release
suites (`fleet.json`): pull requests run vet and tests on Linux and Windows, a merge
that changes the code takes the next `v0.1.N` tag, and gvoice-tui and at-claude take
it up through their `go.dependencies`.
