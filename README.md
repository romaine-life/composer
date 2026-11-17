# composer

The message box shared by Nelson's terminal apps, gvoice-tui and at-claude: one place
for the text, the cursor, wrapping and every editing key, so a change to how typing
works reaches both. Work package:
[APPS-67](https://openproject.home.romaine.life/wp/APPS-67).

It is a Go library with no terminal of its own, like fzt: an app passes it keys and
pastes and draws it where it wants.

## Shipping

Forgejo is its home; GitHub gets a one-way copy so `go` can fetch it by its
`github.com/romaine-life/composer` name. CI is pipeline-templates' go and release
suites (`fleet.json`): a merge that changes the code takes the next `v0.1.N` tag, and
the apps take it up through their `go.dependencies`.
