Output docs/demos/query-builder.gif

Set Shell "zsh"
Set FontSize 13
Set Width 1400
Set Height 850
Set Theme "Dracula"
Set Padding 10
Set TypingSpeed 50ms

Env HOME "/tmp/qraqula-demo-home"

# Launch the app
Hide
Type "./qla"
Enter
Sleep 2s
Show

# --- 1. Set endpoint and fetch schema ---
Ctrl+K
Sleep 300ms
Type "https://countries.trevorblades.com/graphql"
Sleep 500ms
Ctrl+J
Sleep 300ms
Ctrl+R
Sleep 4s

# --- 2. Open query builder from editor ---
Enter
Sleep 800ms

# --- 3. Pick "countries" operation ---
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms
Enter
Sleep 800ms

# --- 4. Navigate and select fields ---
# Select "code" (first field, already on it)
Type " "
Sleep 400ms

# Move to "name" and select it
Type "j"
Sleep 200ms
Type " "
Sleep 400ms

# Move to "capital" and select it
Type "j"
Sleep 200ms
Type " "
Sleep 400ms

# Move to "emoji" and select it
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms
Type " "
Sleep 400ms

# --- 5. Expand "continent" to see nested fields ---
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms
Type "l"
Sleep 600ms

# Select continent > name
Type "j"
Sleep 200ms
Type " "
Sleep 600ms

# Go back up
Type "h"
Sleep 400ms

# --- 6. Use / to filter fields ---
Type "/"
Sleep 500ms
Type "lang"
Sleep 800ms

# Select "languages" from filtered results
Type " "
Sleep 400ms

# Expand it
Type "l"
Sleep 600ms

# Select "name" inside languages
Type "j"
Sleep 200ms
Type " "
Sleep 600ms

# --- 7. Clear filter with Esc ---
Escape
Sleep 800ms

# --- 8. Scroll through the tree to review selections ---
Type "g"
Sleep 300ms
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 800ms

# --- 9. Apply the built query ---
Alt+Enter
Sleep 800ms

# --- 10. Execute the query ---
Alt+Enter
Sleep 4s

# --- 11. Scroll results ---
Ctrl+L
Sleep 300ms
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 800ms

# --- Final ---
Sleep 2s
