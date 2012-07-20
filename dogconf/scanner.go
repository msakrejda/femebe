// The same as the standard Scanner, except with different scanning
// rules tailored for the Dogs configuration language.  These
// alterations bear the following copyright:
//
// Copyright 2012 Heroku. All rights reserved.
//
// Otherwise, the copyright is:
//
// Copyright 2009 The Go Authors. All rights reserved.  Use of this
// source code is governed by a BSD-style license that can be found in
// the LICENSE file.

package dogconf

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"unicode"
	"unicode/utf8"
)

// A source position is represented by a Position value.
// A position is valid if Line > 0.
type Position struct {
	Filename string // filename, if any
	Offset   int    // byte offset, starting at 0
	Line     int    // line number, starting at 1
	Column   int    // column number, starting at 1 (character count per line)
}

// IsValid returns true if the position is valid.
func (pos *Position) IsValid() bool { return pos.Line > 0 }

func (pos Position) String() string {
	s := pos.Filename
	if pos.IsValid() {
		if s != "" {
			s += ":"
		}
		s += fmt.Sprintf("%d:%d", pos.Line, pos.Column)
	}
	if s == "" {
		s = "???"
	}
	return s
}

type TokenType int16

// The result of Scan is one of the following tokens or a Unicode character.
const (
	EOF TokenType = iota
	LBrace
	RBrace
	Equals
	At
	Comma
	Ident
	Int
	String
)

var tokenString = map[TokenType]string{
	EOF:    "EOF",
	LBrace: "[",
	RBrace: "]",
	Equals: "=",
	At:     "@",
	Comma:  ",",
	Ident:  "Ident",
	Int:    "Int",
	String: "String",
}

func TokenTypeStr(tokTyp TokenType) string {
	return tokenString[tokTyp]
}

// position, lexeme, type in token; no keywords
type Token struct {
	Lexeme string
	Type   TokenType
	Pos    Position
}

func (t *Token) String() string {
	typStr := tokenString[t.Type]
	if typStr == t.Lexeme {
		return fmt.Sprintf("%v at %v", t.Lexeme, t.Pos)
	} else {
		return fmt.Sprintf("%v %v at %v", typStr, t.Lexeme, t.Pos)
	}
	panic("Oh snap!")

}

func IsWhitespace(ch rune) bool {
	return ch == '\t' || ch == '\n' || ch == '\r' || ch == ' '
}

const bufLen = 1024 // at least utf8.UTFMax

// A Scanner implements reading of Unicode characters and tokens from an io.Reader.
type Scanner struct {
	// Input
	src io.Reader

	// Source buffer
	srcBuf [bufLen + 1]byte // +1 for sentinel for common case of s.next()
	srcPos int              // reading position (srcBuf index)
	srcEnd int              // source end (srcBuf index)

	// Source position
	srcBufOffset int // byte offset of srcBuf[0] in source
	line         int // line count
	column       int // character count
	lastLineLen  int // length of last line in characters (for correct column reporting)
	lastCharLen  int // length of last character in bytes

	// Token text buffer
	// Typically, token text is stored completely in srcBuf, but in general
	// the token text's head may be buffered in tokBuf while the token text's
	// tail is stored in srcBuf.
	tokBuf bytes.Buffer // token text head that is not in srcBuf anymore
	tokPos int          // token text tail position (srcBuf index); valid if >= 0
	tokEnd int          // token text tail end (srcBuf index)

	// Next token to be returned from Scan, if pre-scanned for Peek
	nextTok *Token

	// One character look-ahead
	ch rune // character before current srcPos

	// Error is called for each error encountered. If no Error
	// function is set, the error is reported to os.Stderr.
	Error func(s *Scanner, msg string)

	// ErrorCount is incremented by one for each error encountered.
	ErrorCount int

	// Start position of most recently scanned token; set by Scan.
	// Calling Init or Next invalidates the position (Line == 0).
	// The Filename field is always left untouched by the Scanner.
	// If an error is reported (via Error) and Position is invalid,
	// the scanner is not inside a token. Call Pos to obtain an error
	// position in that case.
	Position
}

// Init initializes a Scanner with a new source and returns s.
// Error is set to nil, ErrorCount is set to 0, Mode is set to GoTokens,
// and Whitespace is set to GoWhitespace.
func (s *Scanner) Init(src io.Reader) *Scanner {
	s.src = src

	// initialize source buffer
	// (the first call to next() will fill it by calling src.Read)
	s.srcBuf[0] = utf8.RuneSelf // sentinel
	s.srcPos = 0
	s.srcEnd = 0

	// initialize source position
	s.srcBufOffset = 0
	s.line = 1
	s.column = 0
	s.lastLineLen = 0
	s.lastCharLen = 0

	// initialize token text buffer
	// (required for first call to next()).
	s.tokPos = -1

	// Next token to scan
	s.nextTok = nil

	// initialize one character look-ahead
	s.ch = -1 // no char read yet

	// initialize public fields
	s.Error = nil
	s.ErrorCount = 0

	s.Line = 0 // invalidate token position

	return s
}

// next reads and returns the next Unicode character. It is designed such
// that only a minimal amount of work needs to be done in the common ASCII
// case (one test to check for both ASCII and end-of-buffer, and one test
// to check for newlines).
func (s *Scanner) next() rune {
	ch, width := rune(s.srcBuf[s.srcPos]), 1

	if ch >= utf8.RuneSelf {
		// uncommon case: not ASCII or not enough bytes
		for s.srcPos+utf8.UTFMax > s.srcEnd && !utf8.FullRune(s.srcBuf[s.srcPos:s.srcEnd]) {
			// not enough bytes: read some more, but first
			// save away token text if any
			if s.tokPos >= 0 {
				s.tokBuf.Write(s.srcBuf[s.tokPos:s.srcPos])
				s.tokPos = 0
				// s.tokEnd is set by Scan()
			}
			// move unread bytes to beginning of buffer
			copy(s.srcBuf[0:], s.srcBuf[s.srcPos:s.srcEnd])
			s.srcBufOffset += s.srcPos
			// read more bytes
			// (an io.Reader must return io.EOF when it reaches
			// the end of what it is reading - simply returning
			// n == 0 will make this loop retry forever; but the
			// error is in the reader implementation in that case)
			i := s.srcEnd - s.srcPos
			n, err := s.src.Read(s.srcBuf[i:bufLen])
			s.srcPos = 0
			s.srcEnd = i + n
			s.srcBuf[s.srcEnd] = utf8.RuneSelf // sentinel
			if err != nil {
				if s.srcEnd == 0 {
					if s.lastCharLen > 0 {
						// previous character was not EOF
						s.column++
					}
					s.lastCharLen = 0
					return -1
				}
				if err != io.EOF {
					s.error(err.Error())
				}
				// If err == EOF, we won't be getting more
				// bytes; break to avoid infinite loop. If
				// err is something else, we don't know if
				// we can get more bytes; thus also break.
				break
			}
		}
		// at least one byte
		ch = rune(s.srcBuf[s.srcPos])
		if ch >= utf8.RuneSelf {
			// uncommon case: not ASCII
			ch, width = utf8.DecodeRune(s.srcBuf[s.srcPos:s.srcEnd])
			if ch == utf8.RuneError && width == 1 {
				// advance for correct error position
				s.srcPos += width
				s.lastCharLen = width
				s.column++
				s.error("illegal UTF-8 encoding")
				return ch
			}
		}
	}

	// advance
	s.srcPos += width
	s.lastCharLen = width
	s.column++

	// special situations
	switch ch {
	case 0:
		// for compatibility with other tools
		s.error("illegal character NUL")
	case '\n':
		s.line++
		s.lastLineLen = s.column
		s.column = 0
	}

	return ch
}

func (s *Scanner) peek() rune {
	// Peek returns the next Unicode character in the source
	// without advancing the scanner. It returns EOF if the
	// scanner's position is at the last character of the source.
	if s.ch < 0 {
		s.ch = s.next()
	}
	return s.ch
}

func (s *Scanner) Peek() *Token {
	// TODO: EOF sort of works, but should be revisited
	if s.nextTok == nil {
		s.nextTok = s.Scan()
	}
	return s.nextTok
}

func (s *Scanner) error(msg string) {
	s.ErrorCount++
	if s.Error != nil {
		s.Error(s, msg)
	} else {
		pos := s.Position
		if !pos.IsValid() {
			pos = s.Pos()
		}
		fmt.Fprintf(os.Stderr, "%s: %s\n", pos, msg)
	}

}

func (s *Scanner) scanIdentifier() rune {
	ch := s.next() // read character after first '_' or letter
	for ch == '_' || unicode.IsLetter(ch) || unicode.IsDigit(ch) {
		ch = s.next()
	}
	return ch
}

func digitVal(ch rune) int {
	switch {
	case '0' <= ch && ch <= '9':
		return int(ch - '0')
	case 'a' <= ch && ch <= 'f':
		return int(ch - 'a' + 10)
	case 'A' <= ch && ch <= 'F':
		return int(ch - 'A' + 10)
	}
	return 16 // larger than any legal digit val
}

func isDecimal(ch rune) bool { return '0' <= ch && ch <= '9' }

func (s *Scanner) scanInt(ch rune) rune {
	for isDecimal(ch) {
		ch = s.next()
	}
	return ch
}

func (s *Scanner) scanNumber(ch rune) rune {
	// isDecimal(ch)
	if ch == '0' {
		// int or float
		ch = s.next()
		if ch == 'x' || ch == 'X' {
			// hexadecimal int
			ch = s.next()
			for digitVal(ch) < 16 {
				ch = s.next()
			}
		} else {
			// octal int or float
			seenDecimalDigit := false
			for isDecimal(ch) {
				if ch > '7' {
					seenDecimalDigit = true
				}
				ch = s.next()
			}
			// octal int
			if seenDecimalDigit {
				s.error("illegal octal number")
			}
		}
		return ch
	}
	// decimal int or float
	return s.scanInt(ch)
}

func (s *Scanner) scanString(quote rune) rune {
redo:
	ch := s.next() // read character after quote
	for ch != quote {
		if ch < 0 {
			s.error("literal not terminated")
			return -1
		}
		ch = s.next()
	}
	ch = s.next()
	if ch == quote {
		goto redo
	}
	return ch
}

// Scan reads the next Token from source and returns it.  It only
// recognizes the built-in tokens. It returns nil at the end of the
// source. It reports scanner errors (read and token errors) by
// calling s.Error, if not nil; otherwise it prints an error message
// to os.Stderr.
func (s *Scanner) Scan() *Token {
	if s.nextTok != nil {
		result := s.nextTok
		s.nextTok = nil
		return result
	}

	ch := s.peek()

	// reset token text position
	s.tokPos = -1
	s.Line = 0

	// skip white space
	for IsWhitespace(ch) {
		ch = s.next()
	}

	if ch == -1 {
		return &Token{"EOF", EOF, s.Pos()}
	}

	// start collecting token text
	s.tokBuf.Reset()
	s.tokPos = s.srcPos - s.lastCharLen

	// set token position
	// (this is a slightly optimized version of the code in Pos())
	s.Offset = s.srcBufOffset + s.tokPos
	if s.column > 0 {
		// common case: last character was not a '\n'
		s.Line = s.line
		s.Column = s.column
	} else {
		// last character was a '\n'
		// (we cannot be at the beginning of the source
		// since we have called next() at least once)
		s.Line = s.line - 1
		s.Column = s.lastLineLen
	}

	// determine token value
	var tokTyp TokenType
	switch {
	case unicode.IsLetter(ch) || ch == '_':
		tokTyp = Ident
		ch = s.scanIdentifier()
	case isDecimal(ch):
		tokTyp = Int
		ch = s.scanNumber(ch)
	default:
		switch ch {
		case '\'':
			// we don't call s.next() after parsing a string,
			// because we need more lookahead in this case to account
			// for quote-escaped quoting
			ch = s.scanString('\'')
			tokTyp = String
		case '[':
			tokTyp = LBrace
			ch = s.next()
		case ']':
			tokTyp = RBrace
			ch = s.next()
		case '=':
			tokTyp = Equals
			ch = s.next()
		case '@':
			tokTyp = At
			ch = s.next()
		case ',':
			tokTyp = Comma
			ch = s.next()
		default:
			// TODO: token not recognized by lexer, return error
			fmt.Errorf("Error, unexpected token %v", ch)
			tokTyp = Ident
			ch = s.next()
		}
	}

	// end of token text
	s.tokEnd = s.srcPos - s.lastCharLen

	s.ch = ch

	return &Token{s.tokenText(), tokTyp, s.Pos()}
}

// Pos returns the position of the character immediately after
// the character or token returned by the last call to Next or Scan.
func (s *Scanner) Pos() (pos Position) {
	pos.Filename = s.Filename
	pos.Offset = s.srcBufOffset + s.srcPos - s.lastCharLen
	switch {
	case s.column > 0:
		// common case: last character was not a '\n'
		pos.Line = s.line
		pos.Column = s.column
	case s.lastLineLen > 0:
		// last character was a '\n'
		pos.Line = s.line - 1
		pos.Column = s.lastLineLen
	default:
		// at the beginning of the source
		pos.Line = 1
		pos.Column = 1
	}
	return
}

// TokenText returns the string corresponding to the most recently scanned token.
// Valid after calling Scan().
func (s *Scanner) tokenText() string {
	if s.tokPos < 0 {
		// no token text
		return ""
	}

	if s.tokEnd < 0 {
		// if EOF was reached, s.tokEnd is set to -1 (s.srcPos == 0)
		s.tokEnd = s.tokPos
	}

	if s.tokBuf.Len() == 0 {
		// common case: the entire token text is still in srcBuf
		return string(s.srcBuf[s.tokPos:s.tokEnd])
	}

	// part of the token text was saved in tokBuf: save the rest in
	// tokBuf as well and return its content
	s.tokBuf.Write(s.srcBuf[s.tokPos:s.tokEnd])
	s.tokPos = s.tokEnd // ensure idempotency of TokenText() call
	return s.tokBuf.String()
}
