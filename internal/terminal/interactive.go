package terminal

import (
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

// ANSI color and style codes
const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Dim       = "\033[2m"
	Italic    = "\033[3m"
	Underline = "\033[4m"

	// Colors
	Black   = "\033[30m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"

	// Bright colors
	BrightBlack   = "\033[90m"
	BrightRed     = "\033[91m"
	BrightGreen   = "\033[92m"
	BrightYellow  = "\033[93m"
	BrightBlue    = "\033[94m"
	BrightMagenta = "\033[95m"
	BrightCyan    = "\033[96m"
	BrightWhite   = "\033[97m"

	// Terminal control
	ClearScreen  = "\033[2J"
	ClearLine    = "\033[2K"
	CursorHome   = "\033[H"
	HideCursor   = "\033[?25l"
	ShowCursor   = "\033[?25h"
	AltScreenOn  = "\033[?1049h"
	AltScreenOff = "\033[?1049l"
)

// TermState holds the terminal state for raw mode
type TermState struct {
	fd       int
	oldState *term.State
}

// MakeRaw sets terminal to raw mode and returns the old state
func MakeRaw() (*TermState, error) {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}

	return &TermState{fd: fd, oldState: oldState}, nil
}

// Restore restores terminal to previous state
func (t *TermState) Restore() {
	if t != nil && t.oldState != nil {
		_ = term.Restore(t.fd, t.oldState)
	}
}

// ReadKey reads a single key press and returns a string representation
func ReadKey() string {
	var buf [3]byte
	n, err := os.Stdin.Read(buf[:1])
	if err != nil || n == 0 {
		return ""
	}

	// Check for escape sequence (arrow keys)
	if buf[0] == 27 { // ESC
		// Try to read more bytes for escape sequence
		os.Stdin.Read(buf[1:3])
		if buf[1] == '[' {
			switch buf[2] {
			case 'A':
				return "UP"
			case 'B':
				return "DOWN"
			case 'C':
				return "RIGHT"
			case 'D':
				return "LEFT"
			}
		}
		return "ESC"
	}

	switch buf[0] {
	case 32: // Space
		return "SPACE"
	case 13, 10: // Enter
		return "ENTER"
	case 127, 8: // Backspace
		return "BACKSPACE"
	case 9: // Tab
		return "TAB"
	default:
		return string(buf[0])
	}
}

// IsInteractiveTerminal checks if stdin is a terminal
func IsInteractiveTerminal() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// WriteLine writes a line with proper carriage return for raw mode
func WriteLine(s string) {
	fmt.Print(s + "\r\n")
}

// Write writes a string without newline
func Write(s string) {
	fmt.Print(s)
}

// Color returns a colored string
func Color(s string, color string) string {
	return color + s + Reset
}

// StatusColor returns the appropriate color for a status
func StatusColor(status string) string {
	switch status {
	case "done":
		return Green
	case "open":
		return Blue
	case "blocked":
		return Red
	case "waiting":
		return Yellow
	case "tech-debt":
		return Magenta
	default:
		return White
	}
}

// StatusIcon returns the appropriate icon for a status
func StatusIcon(status string) string {
	switch status {
	case "done":
		return "✓"
	case "open":
		return "○"
	case "blocked":
		return "✗"
	case "waiting":
		return "◔"
	case "tech-debt":
		return "⚠"
	default:
		return "○"
	}
}

// PrintHeader prints a styled header box
func PrintHeader(title, icon string) {
	const baseWidth = 55 // minimum inner width between vertical borders

	iconWidth := runewidth.StringWidth(icon)
	titleWidth := runewidth.StringWidth(title)
	textWidth := 2 + iconWidth + 2 + titleWidth // spaces after │ and around icon
	innerWidth := baseWidth
	if textWidth > innerWidth {
		innerWidth = textWidth
	}
	padding := innerWidth - textWidth
	bar := strings.Repeat("─", innerWidth)

	fmt.Println()
	fmt.Printf("  %s%s╭%s╮%s\n", Bold, BrightCyan, bar, Reset)
	fmt.Printf("  %s%s│  %s  %s%s│%s\n", Bold, BrightCyan, icon, title, strings.Repeat(" ", padding), Reset)
	fmt.Printf("  %s%s╰%s╯%s\n", Bold, BrightCyan, bar, Reset)
	fmt.Println()
}

// PrintSuccess prints a success message
func PrintSuccess(msg string) {
	fmt.Printf("  %s%s✓ %s%s\n", BrightGreen, Bold, msg, Reset)
}

// PrintError prints an error message
func PrintError(msg string) {
	fmt.Printf("  %s%s✗ %s%s\n", BrightRed, Bold, msg, Reset)
}

// PrintWarning prints a warning message
func PrintWarning(msg string) {
	fmt.Printf("  %s%s⚠ %s%s\n", BrightYellow, Bold, msg, Reset)
}

// PrintInfo prints an info message
func PrintInfo(msg string) {
	fmt.Printf("  %s%sℹ %s%s\n", BrightBlue, Bold, msg, Reset)
}

// PrintDim prints a dimmed message
func PrintDim(msg string) {
	fmt.Printf("  %s%s%s\n", Dim, msg, Reset)
}

// Truncate truncates a string to the given length
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
