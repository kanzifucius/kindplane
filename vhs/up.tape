Source vhs/config.tape
Output docs/assets/vhs/up.gif

Set PlaybackSpeed 6

# Ensure config file exists
Hide
Type "kindplane init --force"
Enter
Sleep 1s
Show

# Create and bootstrap cluster
Type "kindplane up"
Sleep 500ms
Enter
Sleep 180s
Set PlaybackSpeed 1
Sleep 3s
