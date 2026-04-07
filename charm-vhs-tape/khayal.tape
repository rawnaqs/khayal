# khayal.tape — demo recording for khayal.rawnaqs.io
# Font: FiraCodeNerdFont, Theme: Catppuccin Mocha
# Run: vhs khayal.tape

Output khayal.gif
Output khayal.mp4

Set Shell "bash"
Set FontSize 14
Set FontFamily "FiraCode Nerd Font Mono"
Set Theme "Catppuccin Mocha"
Set Width 1200
Set Height 650
Set Padding 24
Set MarginFill "#1e1e2e"
Set Margin 36
Set BorderRadius 12
Set WindowBar Colorful
Set TypingSpeed 40ms
Set PlaybackSpeed 0.8

# --- 1. boot the server ---

Sleep 500ms
Type "brew services start khayal"
Enter
Sleep 3500ms

# --- 2. capture a quick thought ---

Type "kl 'the observer effect changes what you measure'"
Enter
Sleep 2s

# --- 3. capture a URL ---

Type "kl url https://arme.dev/blog/blog"
Enter
Sleep 2s

# --- 6. search ---

Sleep 500ms
Type "kl search 'observer'"
Enter
Sleep 2500ms

# --- 8. stats ---

Type "kl stats"
Enter
Sleep 3s

# hold on the final frame
Sleep 2s
