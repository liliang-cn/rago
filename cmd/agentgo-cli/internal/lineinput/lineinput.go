package lineinput

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

var ErrInputCanceled = errors.New("input canceled")

// ReadInteractiveLine reads one editable terminal line with UTF-8 aware cursor
// movement and deletion. When stdin/stdout are not terminals, it falls back to
// simple line reading.
func ReadInteractiveLine(prompt string) (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return readLineFallback(prompt)
	}

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = term.Restore(fd, oldState)
	}()

	fmt.Print(prompt)

	var (
		buf      []rune
		cursor   int
		byteBuf  = make([]byte, 1)
		lastCols int
	)

	render := func() {
		line := string(buf)
		displayWidth := runewidth.StringWidth(line)
		if displayWidth < lastCols {
			fmt.Print("\r", prompt, line, strings.Repeat(" ", lastCols-displayWidth), "\r", prompt)
		} else {
			fmt.Print("\r", prompt, line, "\r", prompt)
		}
		if cursor > 0 {
			fmt.Print(renderCursorPrefix(string(buf[:cursor])))
		}
		lastCols = displayWidth
	}

	for {
		if _, err := os.Stdin.Read(byteBuf); err != nil {
			return "", err
		}

		switch b := byteBuf[0]; b {
		case '\r', '\n':
			padding := max(0, lastCols-runewidth.StringWidth(string(buf)))
			fmt.Print("\r", prompt, string(buf), strings.Repeat(" ", padding), "\r\n")
			return string(buf), nil
		case 3:
			return "", ErrInputCanceled
		case 4:
			if len(buf) == 0 {
				return "", io.EOF
			}
		case 127, 8:
			if cursor > 0 {
				buf = append(buf[:cursor-1], buf[cursor:]...)
				cursor--
				render()
			}
		case 27:
			seq, err := readEscapeSequence()
			if err != nil {
				return "", err
			}
			switch seq {
			case "[D":
				if cursor > 0 {
					cursor--
					render()
				}
			case "[C":
				if cursor < len(buf) {
					cursor++
					render()
				}
			case "[3~":
				if cursor < len(buf) {
					buf = append(buf[:cursor], buf[cursor+1:]...)
					render()
				}
			case "[H", "OH":
				if cursor != 0 {
					cursor = 0
					render()
				}
			case "[F", "OF":
				if cursor != len(buf) {
					cursor = len(buf)
					render()
				}
			}
		default:
			r, size := DecodeRuneFromReader(os.Stdin, b)
			if size == 0 {
				continue
			}
			buf = append(buf[:cursor], append([]rune{r}, buf[cursor:]...)...)
			cursor++
			render()
		}
	}
}

func readLineFallback(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) && len(line) > 0 {
			fmt.Println()
			return strings.TrimRight(line, "\r\n"), nil
		}
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func readEscapeSequence() (string, error) {
	var seq []byte
	buf := make([]byte, 1)
	for len(seq) < 8 {
		if _, err := os.Stdin.Read(buf); err != nil {
			return "", err
		}
		seq = append(seq, buf[0])
		if (buf[0] >= 'A' && buf[0] <= 'Z') || buf[0] == '~' {
			break
		}
	}
	return string(seq), nil
}

// DecodeRuneFromReader is exported for focused UTF-8 input tests.
func DecodeRuneFromReader(reader io.Reader, first byte) (rune, int) {
	if first < utf8.RuneSelf {
		if first < 32 {
			return 0, 0
		}
		return rune(first), 1
	}

	size := utf8SequenceLength(first)
	if size == 0 {
		return utf8.RuneError, 1
	}
	buf := make([]byte, size)
	buf[0] = first
	for i := 1; i < size; i++ {
		if _, err := reader.Read(buf[i : i+1]); err != nil {
			return utf8.RuneError, 1
		}
	}
	r, n := utf8.DecodeRune(buf)
	if r == utf8.RuneError && n == 1 {
		return utf8.RuneError, 1
	}
	return r, n
}

func utf8SequenceLength(first byte) int {
	switch {
	case first&0xE0 == 0xC0:
		return 2
	case first&0xF0 == 0xE0:
		return 3
	case first&0xF8 == 0xF0:
		return 4
	default:
		return 0
	}
}

func renderCursorPrefix(s string) string {
	width := runewidth.StringWidth(s)
	if width <= 0 {
		return ""
	}
	return fmt.Sprintf("\x1b[%dC", width)
}
