Output demo.gif

Set Shell "zsh"
Set FontSize 13
Set Width 1400
Set Height 850
Set Theme "Dracula"
Set Padding 10
Set TypingSpeed 50ms

Env HOME "/tmp/qraqula-demo-home"

# Launch the app (hide the shell prompt)
Hide
Type "./qla"
Enter
Sleep 2s
Show

# --- 1. Navigate to endpoint and enter URL ---
Ctrl+K
Sleep 300ms
Type "https://countries.trevorblades.com/graphql"
Sleep 800ms

# --- 2. Move to editor and fetch schema ---
Ctrl+J
Sleep 300ms
Ctrl+R
Sleep 4s

# --- 3. Write a query (editor is already in insert mode) ---
Type "{"
Enter
Type "  countries {"
Enter
Type "    name"
Enter
Type "    code"
Enter
Type "    capital"
Enter
Type "    emoji"
Enter
Type "  }"
Enter
Type "}"
Sleep 600ms

# --- 4. Execute the query ---
Alt+Enter
Sleep 4s

# --- 5. Browse results (fold/unfold) ---
Ctrl+L
Sleep 400ms
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
# Fold a node
Type "l"
Sleep 400ms
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "l"
Sleep 400ms
Type "k"
Sleep 150ms
# Unfold
Type "l"
Sleep 600ms

# --- 6. Switch to schema browser ---
Ctrl+D
Sleep 500ms

# Navigate the schema tree
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms
Type "l"
Sleep 400ms
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms
Type "l"
Sleep 500ms

# Search in schema
Type "/"
Sleep 300ms
Type "lang"
Sleep 600ms
Enter
Sleep 400ms
Escape
Sleep 300ms

# Back to root
Type "h"
Sleep 300ms
Type "h"
Sleep 500ms

# --- 7. History sidebar (already visible after auto-save) ---
# Tab from schema browser to history panel
Tab
Sleep 500ms

# Create a folder
Type "N"
Sleep 600ms

# Rename the new folder
Type "r"
Sleep 400ms
Backspace
Backspace
Backspace
Backspace
Backspace
Backspace
Backspace
Backspace
Backspace
Backspace
Type "Favorites"
Enter
Sleep 600ms

# Move the query into the folder
Type "j"
Sleep 300ms
Type "m"
Sleep 800ms

# Close sidebar (focus returns to editor)
Ctrl+B
Sleep 800ms

# --- 8. Environment & Header management ---
Ctrl+E
Sleep 800ms

# Create a new environment
Type "n"
Sleep 500ms
Type "prod"
Sleep 300ms
Enter
Sleep 500ms

# Activate the environment
Enter
Sleep 500ms

# Edit endpoint for the env
Type "e"
Sleep 500ms
Type "https://countries.trevorblades.com/graphql"
Sleep 300ms
Enter
Sleep 500ms

# Switch to headers section and add a header
Tab
Sleep 400ms
Type "a"
Sleep 500ms
Type "Authorization"
Enter
Sleep 400ms
Type "Bearer my-token"
Enter
Sleep 600ms

# Close the overlay — env badge appears
Escape
Sleep 1s

# Cycle env on/off with Ctrl+N
Ctrl+N
Sleep 800ms
Ctrl+N
Sleep 800ms

# --- Final ---
Sleep 2s
