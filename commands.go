package composer

// The box's commands, by the name an app's command palette lists them under.
const (
	NewLine           = "new-line"
	NextParagraph     = "next-paragraph"
	PreviousParagraph = "previous-paragraph"
	Undo              = "undo"
	Redo              = "redo"
)

// Command is one of the box's commands as an app's palette lists it.
type Command struct {
	Name        string
	Description string
	Keys        string // The keys that run it, as the palette names them.
}

// Commands is every command the box has, for an app's palette to list as they are
// and run with Do, so both apps name the same keys for the same thing.
var Commands = []Command{
	{NewLine, "Start a new line in your message", "Shift+Enter, Alt+Enter, Ctrl+J"},
	{NextParagraph, "Move to the start of your message's next line", "Ctrl+Down"},
	{PreviousParagraph, "Move to the start of this line of your message, or the one before", "Ctrl+Up"},
	{Undo, "Undo your last change to your message", "Ctrl+Z"},
	{Redo, "Redo what you undid", "Ctrl+Y"},
}

// Do runs one of the box's commands by name. It says whether the name is one.
func (b *Box) Do(name string) bool {
	return b.do(name) == Handled
}

func (b *Box) do(name string) Result {
	switch name {
	case NewLine:
		b.save(false, 0)
		b.insert("\n")
	case NextParagraph:
		b.nextParagraph()
	case PreviousParagraph:
		b.previousParagraph()
	case Undo:
		b.Undo()
	case Redo:
		b.Redo()
	default:
		return Unhandled
	}
	return Handled
}
