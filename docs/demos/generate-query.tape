Output generate-demo.gif

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

# --- 2. Open schema browser ---
Ctrl+D
Sleep 500ms

# Navigate into Query type
Type "l"
Sleep 400ms

# --- 3. Search for "countries" field ---
Type "/"
Sleep 300ms
Type "countries"
Sleep 600ms
Enter
Sleep 400ms

# --- 4. Generate bare query with g ---
Type "g"
Sleep 1500ms

# --- 5. Execute the generated query ---
Alt+Enter
Sleep 4s

# --- 6. Go back to schema browser for G demo ---
Ctrl+D
Sleep 500ms

# Navigate into Query type again
Type "l"
Sleep 400ms

# Search for "country" (has arguments)
Type "/"
Sleep 300ms
# Clear previous search text
Ctrl+A
Backspace
Type "country"
Sleep 600ms
Enter
Sleep 400ms

# --- 7. View args with v ---
Type "v"
Sleep 1500ms

# Go back
Type "h"
Sleep 400ms

# --- 8. Generate full operation with G ---
Type "G"
Sleep 1500ms

# Show the generated variables
Ctrl+J
Sleep 1500ms

# --- Final ---
Sleep 2s
