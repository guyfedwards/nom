# VHS documentation
#
#   Output <path>.mp4               Create an MP4 output at the given <path>
#   Output <path>.webm              Create a WebM output at the given <path>
#
# Require:
#   Require <string>                Ensure a program is on the $PATH to proceed
#
# Settings:
#   Set FontSize <number>           Set the font size of the terminal
#   Set FontFamily <string>         Set the font family of the terminal
#   Set Height <number>             Set the height of the terminal
#   Set Width <number>              Set the width of the terminal
#   Set LetterSpacing <float>       Set the font letter spacing (tracking)
#   Set LineHeight <float>          Set the font line height
#   Set LoopOffset <float>%         Set the starting frame offset for the GIF loop
#   Set Theme <json|string>         Set the theme of the terminal
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
Output demo2.gif
Set FontSize 32
Set FontFamily "JetBrainsMono Nerd Font"
Set Width 2400
Set Height 1200
Set TypingSpeed 300ms
# Set LoopOffset 50%
Set Shell bash

# Clear alias errors
Ctrl+L
Sleep 500ms

Type "./nom" Sleep 500ms Enter

Sleep 3s

Type "j" Sleep 500ms
Type "j" Sleep 500ms
Type "j" Sleep 500ms
Type "j" Sleep 500ms
Type "j" Sleep 500ms
Type "l" Sleep 500ms
Type "h" Sleep 500ms

Sleep 2s

Enter

Sleep 3s

Type "j" Sleep 500ms
Type "j" Sleep 500ms
Type "j" Sleep 500ms
Type "k" Sleep 500ms
Type "gg"

Type "l" Sleep 1s
Type "l" Sleep 1s
Type "h" Sleep 1s
Type "j" Sleep 1s
Type "j" Sleep 1s
Type "j" Sleep 1s
Type "G" Sleep 1s
Type "gg" Sleep 1s

Type "q" Sleep 1s

Type "/" Sleep 1s
Type "evil" Sleep 3s
Enter
Sleep 2s
Type "q"
Type "m" Sleep 1s
Type "j" Sleep 1s
Type "m" Sleep 1s
Type "j" Sleep 1s
Type "m" Sleep 1s
Type "j" Sleep 1s
Type "M" Sleep 3s
Type "M" Sleep 2s
