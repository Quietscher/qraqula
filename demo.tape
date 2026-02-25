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

# --- 3. Write a query with syntax highlighting ---
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

# --- 5. Search results ---
Ctrl+L
Sleep 400ms
Type "/"
Sleep 300ms
Type "Germany"
Sleep 600ms
# Navigate matches
Type "n"
Sleep 400ms
Type "n"
Sleep 400ms
# Exit search
Escape
Sleep 500ms

# --- 6. Browse results (fold/unfold) ---
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

# --- 7. Switch to schema browser ---
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

# --- 8. History sidebar with timestamps ---
# Tab from schema browser to history panel
Tab
Sleep 500ms

# Create a folder
Type "N"
Sleep 600ms

# Rename the new folder
Type "r"
Sleep 400ms
Backspace 10
Type "Favorites"
Enter
Sleep 600ms

# Move the query into the folder
Type "j"
Sleep 300ms
Type "m"
Sleep 800ms

# --- 9. Run a second query to show timestamps and history ---
Ctrl+J
Sleep 300ms
Escape
Type "ggVGd"
Sleep 200ms
Type "i"
Type "{ continents { name code } }"
Sleep 500ms
Alt+Enter
Sleep 4s

# Show sidebar with multiple entries + timestamps
Ctrl+H
Sleep 1s
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "k"
Sleep 500ms

# --- 10. Delete folder with confirmation ---
# Navigate to folder
Type "k"
Sleep 300ms
# Press d to delete — shows confirmation
Type "d"
Sleep 800ms
# Cancel with n
Type "n"
Sleep 600ms

# Close sidebar
Ctrl+B
Sleep 800ms

# --- 11. Schema-aware linting ---
# Write a query with invalid field
Escape
Type "ggVGd"
Sleep 200ms
Type "i"
Type "{ countries { invalidField } }"
Sleep 600ms
# Linting shows error
Alt+Enter
Sleep 3s

# Fix the query
Escape
Type "ggVGd"
Sleep 200ms
Type "i"
Type "{ countries { name capital } }"
Sleep 600ms
Alt+Enter
Sleep 4s

# --- 12. Environment & Header management ---
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
