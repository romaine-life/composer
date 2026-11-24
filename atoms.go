package composer

import "regexp"

// SetAtoms makes each stretch of the message that pattern matches one unit, an
// atom: the cursor steps over it whole and never stops inside it, Backspace and
// Delete take it whole, a deletion that would cut into it takes all of it, it
// wraps to a row of its own rather than split (unless it is wider than a row), and
// Draw and View show it in Styles.Atom. The text stays the message's own, so Value,
// SetValue, Insert, undo and a draft carry atoms as text: whatever matches the
// pattern is one, however it got there. nil makes none.
//
// The pattern is the app's: at-claude marks the objects picked from its menu
// (ATC-191). It should match nothing that typing next to an atom extends, such as
// a delimited form.
func (b *Box) SetAtoms(pattern *regexp.Regexp) {
	b.atom = pattern
	b.pos = b.snap(b.paras[b.line], b.pos)
	b.reveal()
}

// atomsIn is where the atoms are in a paragraph: byte ranges, in order.
func (b *Box) atomsIn(p string) [][]int {
	if b.atom == nil {
		return nil
	}
	return b.atom.FindAllStringIndex(p, -1)
}

// within is the atom in p that offset i falls strictly inside, if any.
func within(atoms [][]int, i int) (start, end int, ok bool) {
	for _, a := range atoms {
		if a[0] < i && i < a[1] {
			return a[0], a[1], true
		}
	}
	return 0, 0, false
}

// in says the character at offset i is part of an atom.
func in(atoms [][]int, i int) bool {
	for _, a := range atoms {
		if a[0] <= i && i < a[1] {
			return true
		}
	}
	return false
}

// snap moves offset i in p back to the start of the atom it falls inside.
func (b *Box) snap(p string, i int) int {
	if s, _, ok := within(b.atomsIn(p), i); ok {
		return s
	}
	return i
}

// snapEnd moves offset i in p on to the end of the atom it falls inside.
func (b *Box) snapEnd(p string, i int) int {
	if _, e, ok := within(b.atomsIn(p), i); ok {
		return e
	}
	return i
}

// back is the place a step left of offset i in p reaches: a grapheme back, or the
// start of the atom that ends at i.
func (b *Box) back(p string, i int) int {
	return b.snap(p, before(p, i))
}

// forward is the place a step right of offset i in p reaches: a grapheme on, or
// the end of the atom that starts at i.
func (b *Box) forward(p string, i int) int {
	return b.snapEnd(p, after(p, i))
}
