Output schema-browser.gif

Set Shell "zsh"
Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Dracula"

Type "go run ./cmd/qla"
Enter
Sleep 2s

# Type an endpoint URL
Type "https://countries.trevorblades.com/graphql"
Sleep 500ms

# Tab to editor
Type "	"
Sleep 1s

# Toggle to schema browser
Ctrl+D
Sleep 500ms

# Refresh schema
Ctrl+R
Sleep 3s

# Tab to the schema browser panel
Ctrl+L
Sleep 500ms

# Navigate down
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms

# Drill into a type
Type "l"
Sleep 500ms

# Navigate fields
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms

# Drill deeper
Type "l"
Sleep 500ms

# Go back
Type "h"
Sleep 500ms

# Try search
Type "/"
Sleep 300ms
Type "cont"
Sleep 500ms

# Accept search
Enter
Sleep 500ms

# Clear and go back to root
Escape
Sleep 300ms
Type "h"
Sleep 500ms

Sleep 2s
