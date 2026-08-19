# VHS documentation
#
# Run with:
#  vhs < .github/assets/demo.tape
#
# Output:
#   Output <path>.gif               Create a GIF output at the given <path>
#   Output <path>.mp4               Create an MP4 output at the given <path>
#   Output <path>.webm              Create a WebM output at the given <path>
#
# Settings:
#   Set FontSize <number>           Set the font size of the terminal
#   Set FontFamily <string>         Set the font family of the terminal
#   Set Height <number>             Set the height of the terminal
#   Set Width <number>              Set the width of the terminal
#   Set LetterSpacing <float>       Set the font letter spacing (tracking)
#   Set LineHeight <float>          Set the font line height
#   Set Theme <string>              Set the theme of the terminal (JSON)
#   Set Padding <number>            Set the padding of the terminal
#   Set Framerate <number>          Set the framerate of the recording
#   Set PlaybackSpeed <float>       Set the playback speed of the recording
#
# Sleep:
#   Sleep <time>                    Sleep for a set amount of <time> in seconds
#
# Type:
#   Type[@<time>] "<characters>"    Type <characters> into the terminal with a
#                                   <time> delay between each character
#
# Keys:
#   Backspace[@<time>] [number]     Press the Backspace key
#   Down[@<time>] [number]          Press the Down key
#   Enter[@<time>] [number]         Press the Enter key
#   Space[@<time>] [number]         Press the Space key
#   Tab[@<time>] [number]           Press the Tab key
#   Left[@<time>] [number]          Press the Left Arrow key
#   Right[@<time>] [number]         Press the Right Arrow key
#   Up[@<time>] [number]            Press the Up Arrow key
#   Down[@<time>] [number]          Press the Down Arrow key
#   Ctrl+<key>                      Press the Control key + <key> (e.g. Ctrl+C)
#
# Display:
#   Hide                            Hide the subsequent commands from the output
#   Show                            Show the subsequent commands in the output

Output .github/assets/demo.gif
Set Shell bash
Set PlaybackSpeed 1.0
Set TypingSpeed 10ms

Set Theme "Catppuccin Macchiato"
Set FontFamily "Hack Nerd Font Mono"

Set FontSize 14
Set Width 1600
Set Height 1000

Hide
Type `cd ~/Developer/kyleking/gh-sweep`  Enter  Sleep 500ms
Type `go build -o gh-sweep ./cmd/gh-sweep`  Enter  Sleep 2s
Type `clear`  Enter
Show

# Start the TUI, scoped to gh-sweep's own repo
Type `./gh-sweep --repo KyleKing/gh-sweep`  Sleep 500ms  Enter
Sleep 2s

# Home menu: point out the namespace-audit vs per-repo view grouping
Down 3
Sleep 800ms
Up 3
Sleep 700ms

# Branch management: navigate, show the help legend, then back out
Type `1`
Sleep 3s
Down 2
Sleep 700ms
Type `?`
Sleep 2s
Escape
Sleep 700ms
Escape
Sleep 700ms

# Analytics: CI/CD stats, then flip to flaky-test and error tabs
Type `4`
Sleep 2s
Type `2`
Sleep 1.2s
Type `3`
Sleep 1.5s
Escape
Sleep 700ms

# Back to the home menu, then quit
Type `q`
Sleep 300ms
