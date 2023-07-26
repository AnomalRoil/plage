package plage

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"filippo.io/age"
)

// Command allows you to define your own commands and responses
type Command age.Stanza

// Name return the command or response name or type as a string
func (c *Command) Name() string {
	return c.Type
}

// Metadata returns the metadata provided with the command, if any, otherwise nil
func (c *Command) Metadata() []string {
	return c.Args
}

// Data returns the data associated with the command, if any, otherwise nil
func (c *Command) Data() []string {
	return c.Body
}

// The done command is used to terminate phases
var Done = Command{Type: "done"}

// =================================
// The code below is mostly coming from
// github.com/FiloSottile/age/blob/main/internal/format/format.go
// and is copyrighted by The age Authors since it is in an internal
// package that cannot be used by other packages, but there are some
// changes in order to replace Stanzas with Commands
// =================================

var b64 = base64.RawStdEncoding.Strict()

func DecodeString(s string) ([]byte, error) {
	// CR and LF are ignored by DecodeString, but we don't want any malleability.
	if strings.ContainsAny(s, "\n\r") {
		return nil, errors.New(`unexpected newline character`)
	}
	return b64.DecodeString(s)
}

var EncodeToString = b64.EncodeToString

const ColumnsPerLine = 64

const BytesPerLine = ColumnsPerLine / 4 * 3

// NewWrappedBase64Encoder returns a WrappedBase64Encoder that writes to dst.
func NewWrappedBase64Encoder(enc *base64.Encoding, dst io.Writer) *WrappedBase64Encoder {
	w := &WrappedBase64Encoder{dst: dst}
	w.enc = base64.NewEncoder(enc, WriterFunc(w.writeWrapped))
	return w
}

type WriterFunc func(p []byte) (int, error)

func (f WriterFunc) Write(p []byte) (int, error) { return f(p) }

// WrappedBase64Encoder is a standard base64 encoder that inserts an LF
// character every ColumnsPerLine bytes. It does not insert a newline neither at
// the beginning nor at the end of the stream, but it ensures the last line is
// shorter than ColumnsPerLine, which means it might be empty.
type WrappedBase64Encoder struct {
	enc     io.WriteCloser
	dst     io.Writer
	written int
	buf     bytes.Buffer
}

func (w *WrappedBase64Encoder) Write(p []byte) (int, error) { return w.enc.Write(p) }

func (w *WrappedBase64Encoder) Close() error {
	return w.enc.Close()
}

func (w *WrappedBase64Encoder) writeWrapped(p []byte) (int, error) {
	if w.buf.Len() != 0 {
		panic("age: internal error: non-empty WrappedBase64Encoder.buf")
	}
	for len(p) > 0 {
		toWrite := ColumnsPerLine - (w.written % ColumnsPerLine)
		if toWrite > len(p) {
			toWrite = len(p)
		}
		n, _ := w.buf.Write(p[:toWrite])
		w.written += n
		p = p[n:]
		if w.written%ColumnsPerLine == 0 {
			w.buf.Write([]byte("\n"))
		}
	}
	if _, err := w.buf.WriteTo(w.dst); err != nil {
		// We always return n = 0 on error because it's hard to work back to the
		// input length that ended up written out. Not ideal, but Write errors
		// are not recoverable anyway.
		return 0, err
	}
	return len(p), nil
}

// LastLineIsEmpty returns whether the last output line was empty, either
// because no input was written, or because a multiple of BytesPerLine was.
//
// Calling LastLineIsEmpty before Close is meaningless.
func (w *WrappedBase64Encoder) LastLineIsEmpty() bool {
	return w.written%ColumnsPerLine == 0
}

var stanzaPrefix = []byte("->")

func (r *Command) Marshal(w io.Writer) error {
	if _, err := w.Write(stanzaPrefix); err != nil {
		return err
	}
	for _, a := range append([]string{r.Type}, r.Args...) {
		if _, err := io.WriteString(w, " "+a); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return err
	}
	ww := NewWrappedBase64Encoder(b64, w)
	if _, err := ww.Write(r.Body); err != nil {
		return err
	}
	if err := ww.Close(); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}

type CmdReader struct {
	r   *bufio.Reader
	err error
}

func NewCmdReader(r *bufio.Reader) *CmdReader {
	return &CmdReader{r: r}
}

func (r *CmdReader) ReadCommand() (s *Command, err error) {
	// Read errors are unrecoverable.
	if r.err != nil {
		return nil, r.err
	}
	defer func() { r.err = err }()

	s = &Command{}

	line, err := r.r.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read line: %w", err)
	}
	if !bytes.HasPrefix(line, stanzaPrefix) {
		return nil, fmt.Errorf("malformed stanza opening line: %q", line)
	}
	prefix, args := splitArgs(line)
	if prefix != string(stanzaPrefix) || len(args) < 1 {
		return nil, fmt.Errorf("malformed stanza: %q", line)
	}
	for _, a := range args {
		if !isValidString(a) {
			return nil, fmt.Errorf("malformed stanza: %q", line)
		}
	}
	s.Type = args[0]
	s.Args = args[1:]

	for {
		line, err := r.r.ReadBytes('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read line: %w", err)
		}

		b, err := DecodeString(strings.TrimSuffix(string(line), "\n"))
		if err != nil {
			return nil, errorf("malformed body line %q: %v", line, err)
		}
		if len(b) > BytesPerLine {
			return nil, errorf("malformed body line %q: too long", line)
		}
		s.Body = append(s.Body, b...)
		if len(b) < BytesPerLine {
			// A stanza body always ends with a short line.
			return s, nil
		}
	}
}

type ParseError struct {
	err error
}

func (e *ParseError) Error() string {
	return "parsing command header: " + e.err.Error()
}

func (e *ParseError) Unwrap() error {
	return e.err
}

func errorf(format string, a ...interface{}) error {
	return &ParseError{fmt.Errorf(format, a...)}
}

func splitArgs(line []byte) (string, []string) {
	l := strings.TrimSuffix(string(line), "\n")
	parts := strings.Split(l, " ")
	return parts[0], parts[1:]
}

func isValidString(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if c < 33 || c > 126 {
			return false
		}
	}
	return true
}
