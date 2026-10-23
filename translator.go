package fixjson

import (
	"bytes"
	"strings"
)

var (
	quotEntity = []byte("&quot;")
	lineEnd    = []byte("\n")
	blockEnd   = []byte("*/")
	literals   = [][]byte{[]byte("true"), []byte("false"), []byte("null")}

	// plain characters are copied to the output as is, without any special handling;
	// a character handled by a branch in translate must be excluded here, or the branch never runs
	plainInString, plainOutside [256]bool
)

func init() {
	for ch := range 256 {
		plainInString[ch] = !strings.ContainsRune("\"\\&\t\r\n", rune(ch))
		plainOutside[ch] = !strings.ContainsRune("\"\\&{}[],#/", rune(ch))
	}
}

func translate(src []byte) []byte {
	var (
		escaped            bool
		quote              bool
		inHTMLQuotedString bool
		inEmbeddedQuote    bool
		requireNewLine     bool
		keyString          bool   // the last string literal was opened where an object key is expected
		stack              []byte // to track { and [
	)

	sc := &scanner{src: src}
	dst := make([]byte, 0, len(src)+16)
	for i := 0; i < len(src); i++ {
		// copy a run of plain characters at once
		if !escaped {
			plain := &plainOutside
			if quote {
				plain = &plainInString
			}
			j := i
			for j < len(src) && plain[src[j]] {
				j++
			}
			if j > i {
				dst = append(dst, src[i:j]...)
				i = j - 1
				continue
			}
		}

		ch := src[i]

		// always add escape character and escaped character
		if ch == '\\' || escaped {
			if !escaped && quote { // start of escape sequence in quotes
				if i+1 < len(src) && src[i+1] == '\'' {
					continue // skip escape character
				}
			}

			if esc, ok := controlEscape(ch); ok && escaped && quote { // escaped control character, replace with its escape sequence
				requireNewLine = requireNewLine || ch == '\n'
				ch = esc
			}

			dst = append(dst, ch)
			escaped = !escaped
			continue
		}

		// find html escaped quote character (&quot;)
		if ch == '&' && (!quote || inHTMLQuotedString) && bytes.HasPrefix(src[i:], quotEntity) {
			ch = '"'                 // replace with quote
			i += len(quotEntity) - 1 // go to last char of sequence
			inHTMLQuotedString = !inHTMLQuotedString
		}

		// start or end of string literal
		if ch == '"' {
			if !quote { // start of string literal
				quote = true
				last := lastNonSpace(dst)
				keyString = last == '{' || last == ','
				dst = append(dst, ch)
				continue
			}

			// possible end of string literal
			next, pos := sc.nextAfterQuote(i + 1)
			if inEmbeddedQuote || pos == i+1 { // inside escaping or no spaces between quote and text, it may need escaping
				closes := expectedAfterQuote(next) && (next != ',' || sc.plausibleValueAt(pos+1) || !sc.closesLater(pos+1))
				if !closes {
					dst = append(dst, '\\', '"')
					inEmbeddedQuote = !inEmbeddedQuote // start or continue of escaping
					continue
				}
				inEmbeddedQuote = false
			}

			// end of string literal
			quote = false
			inHTMLQuotedString = false
			dst = append(dst, ch)

			if next == '"' { // check if comma is missing
				dst = append(dst, ',')
			}

			if requireNewLine { // add new line that was previously skipped
				if i+1 < len(src) && src[i+1] == ',' { // after comma, if comma follows string literal
					if !sc.isTrailingComma(i + 1) {
						dst = append(dst, ',')
					}
					i++
				}
				dst = append(dst, '\n')
				requireNewLine = false
			}
			continue
		}

		// inside string literal
		if quote {
			if esc, ok := controlEscape(ch); ok { // escape tabs, returns and new lines
				dst = append(dst, '\\', esc)
				requireNewLine = requireNewLine || ch == '\n'
			} else {
				dst = append(dst, ch)
			}
			continue
		}

		// Track brackets for truncated JSON repair
		switch ch {
		case '{', '[':
			stack = append(stack, ch)
		case '}', ']':
			if len(stack) > 0 {
				last := stack[len(stack)-1]
				if (ch == '}' && last == '{') || (ch == ']' && last == '[') {
					stack = stack[:len(stack)-1]
				}
			}
		}

		// start of inline comment
		comment := commentAt(src, i)
		if comment == lineComment {
			i = sc.skipLineComment(i)
			// keep the newline formatting
			if i < len(src) && src[i] == '\n' {
				if i > 0 && src[i-1] == '\r' { // add carriage return if needed
					dst = append(dst, '\r', '\n')
				} else {
					dst = append(dst, '\n')
				}
			}
			continue
		}

		// start of multi-line comment
		if comment == blockComment {
			i = sc.skipBlockComment(i)
			continue
		}

		// remove trailing commas
		if ch == ',' && sc.isTrailingComma(i) {
			continue // skip comma
		}

		// seems to be a normal character, add it to the output
		dst = append(dst, ch)
	}

	// repair truncated input
	if quote {
		if escaped { // drop dangling escape character, it would escape the closing quote
			dst = dst[:len(dst)-1]
		} else {
			dst = trimPartialEscape(dst)
		}
		dst = append(dst, '"')
	}
	if len(stack) > 0 {
		dst = completeValue(dst, stack[len(stack)-1], keyString)
	}
	for j := len(stack) - 1; j >= 0; j-- {
		switch stack[j] {
		case '{':
			dst = append(dst, '}')
		case '[':
			dst = append(dst, ']')
		}
	}

	return dst
}

func expectedAfterQuote(ch byte) bool {
	return ch == ':' || isValueTerminator(ch)
}

// isValueTerminator reports whether ch can follow a complete JSON value (0 is the end of input).
func isValueTerminator(ch byte) bool {
	return ch == ',' || ch == ']' || ch == '}' || ch == 0
}

// trimPartialEscape drops a \u escape sequence cut off by the end of input, it is invalid in a closed string.
func trimPartialEscape(dst []byte) []byte {
	i := len(dst)
	for i > 0 && len(dst)-i < 4 && isHexDigit(dst[i-1]) {
		i--
	}
	if len(dst)-i == 4 || i < 2 || dst[i-1] != 'u' || dst[i-2] != '\\' {
		return dst
	}
	slashes := 0 // the backslash starts an escape sequence only if it is not escaped itself
	for j := i - 2; j >= 0 && dst[j] == '\\'; j-- {
		slashes++
	}
	if slashes%2 == 0 {
		return dst
	}
	return dst[:i-2]
}

func isHexDigit(ch byte) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

// controlEscape returns the escape letter for a control character not allowed raw in a string literal.
func controlEscape(ch byte) (byte, bool) {
	switch ch {
	case '\t':
		return 't', true
	case '\r':
		return 'r', true
	case '\n':
		return 'n', true
	}
	return 0, false
}

// scanner looks ahead in src. It remembers where the last skipped comments end, so that lookaheads
// repeated from many quotes inside the same comment text take linear time in total.
type scanner struct {
	src         []byte
	line, block commentEnd
}

// commentEnd caches where the last searched comment ends.
type commentEnd struct {
	from, end int
}

// isTrailingComma reports whether the comma at src[i] is followed by a closing bracket or the end of input.
func (sc *scanner) isTrailingComma(i int) bool {
	next, _ := sc.nextNonSpace(i + 1)
	return next == '}' || next == ']' || next == 0
}

// plausibleValueAt reports whether a JSON value or a closing bracket follows src[begin:].
// It is used to tell a comma that separates values from a comma in text with unescaped quotes.
func (sc *scanner) plausibleValueAt(begin int) bool {
	src := sc.src
	ch, pos := sc.nextNonSpace(begin)
	switch {
	case ch == 0, ch == '"', ch == '{', ch == '[', ch == '}', ch == ']':
		return true
	case bytes.HasPrefix(src[pos:], quotEntity):
		return true
	case ch == '-' || (ch >= '0' && ch <= '9'):
		for pos < len(src) && isNumberChar(src[pos]) {
			pos++
		}
	case ch == 't' || ch == 'f' || ch == 'n':
		n := literalLen(src[pos:])
		if n == 0 {
			return false
		}
		pos += n
	default:
		return false
	}
	next, _ := sc.nextNonSpace(pos)
	return isValueTerminator(next)
}

// literalLen returns the length of the true, false or null literal b starts with, or 0.
func literalLen(b []byte) int {
	for _, lit := range literals {
		if bytes.HasPrefix(b, lit) {
			return len(lit)
		}
	}
	return 0
}

func isNumberChar(ch byte) bool {
	return (ch >= '0' && ch <= '9') || ch == '-' || ch == '+' || ch == '.' || ch == 'e' || ch == 'E'
}

// completeValue finishes a key or value cut off by the end of input, so the open brackets can be closed.
// open is the innermost unclosed bracket, keyString tells whether the last string literal is in a key position.
func completeValue(dst []byte, open byte, keyString bool) []byte {
	dst = completeWord(dst, open)
	switch lastNonSpace(dst) {
	case ':':
		return append(dst, "null"...)
	case ',': // the value after the comma was dropped
		return dst[:bytes.LastIndexByte(dst, ',')]
	case '"':
		if open == '{' && keyString {
			return append(dst, ":null"...)
		}
	}
	return dst
}

// completeWord finishes a literal or a number cut off by the end of input, and drops a cut off unquoted key.
func completeWord(dst []byte, open byte) []byte {
	end := len(dst)
	for end > 0 && dst[end-1] <= ' ' {
		end--
	}
	start := end
	for start > 0 && isWordChar(dst[start-1]) {
		start--
	}
	if start == end {
		return dst
	}

	prev := lastNonSpace(dst[:start])
	if open == '{' && (prev == '{' || prev == ',') { // unquoted key, it cannot be completed
		return dst[:start]
	}
	if prev != ':' && prev != ',' && prev != '[' {
		return dst
	}

	word := dst[start:end]
	if word[0] == '-' || (word[0] >= '0' && word[0] <= '9') { // drop an unfinished fraction or exponent
		return dst[:start+len(bytes.TrimRight(word, ".eE+-"))]
	}
	for _, lit := range literals {
		if len(word) < len(lit) && bytes.HasPrefix(lit, word) {
			return append(dst[:end], lit[len(word):]...)
		}
	}
	return dst
}

func isWordChar(ch byte) bool {
	return isNumberChar(ch) || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

// lastNonSpace returns the last character of dst that is not a whitespace, or 0 if there is none.
func lastNonSpace(dst []byte) byte {
	for j := len(dst) - 1; j >= 0; j-- {
		if dst[j] > ' ' {
			return dst[j]
		}
	}
	return 0
}

// nextAfterQuote is like nextNonSpace, but a character right after the quote is returned as is,
// unless it starts a comment. A block comment always counts as one. A line comment counts as one
// when its marker is followed by a whitespace, or when no quote on the rest of the line can close the string,
// so `"//cdn.com/x">` and `"#42" fixed` are taken as text of the string literal.
func (sc *scanner) nextAfterQuote(begin int) (ch byte, pos int) {
	src := sc.src
	if begin >= len(src) || src[begin] <= ' ' {
		return sc.nextNonSpace(begin)
	}
	switch commentAt(src, begin) {
	case blockComment:
		return sc.nextNonSpace(begin)
	case lineComment:
		text := begin + 1 // first character after the comment marker
		if src[begin] == '/' {
			text++
		}
		if text >= len(src) || src[text] <= ' ' || bytes.IndexByte(src[text:sc.skipLineComment(begin)], '"') < 0 {
			return sc.nextNonSpace(begin)
		}
	}
	return src[begin], begin
}

// closesLater reports whether the next quote in src[begin:] can end a string literal.
// A quote followed by a comma and text is only taken as text when a later quote can close the string,
// otherwise the rest of the input would be pulled into the string literal.
func (sc *scanner) closesLater(begin int) bool {
	src := sc.src
	for i := begin; i < len(src); i++ {
		switch src[i] {
		case '\\':
			i++ // skip escaped character
		case '"':
			next, _ := sc.nextAfterQuote(i + 1)
			return expectedAfterQuote(next)
		}
	}
	return false
}

type commentKind int8

const (
	noComment commentKind = iota
	lineComment
	blockComment
)

// commentAt reports which kind of comment starts at src[i].
func commentAt(src []byte, i int) commentKind {
	switch {
	case src[i] == '#':
		return lineComment
	case src[i] != '/' || i+1 >= len(src):
		return noComment
	case src[i+1] == '/':
		return lineComment
	case src[i+1] == '*':
		return blockComment
	}
	return noComment
}

// nextNonSpace finds the next character that is not a whitespace or comment.
// It returns the character and its position, or 0 and len(src) at the end of input.
func (sc *scanner) nextNonSpace(begin int) (ch byte, pos int) {
	src := sc.src
	for i := begin; i < len(src); {
		if src[i] <= ' ' {
			i++
			continue
		}
		switch commentAt(src, i) {
		case lineComment:
			i = sc.skipLineComment(i)
			continue
		case blockComment:
			i = sc.skipBlockComment(i) + 1
			continue
		}
		return src[i], i
	}
	return 0, len(src)
}

// skipLineComment returns the position of the new line that ends the comment starting at src[i].
func (sc *scanner) skipLineComment(i int) int {
	return sc.line.find(sc.src, i+1, lineEnd)
}

// skipBlockComment returns the position of the last character of the comment starting at src[i].
func (sc *scanner) skipBlockComment(i int) int {
	return min(sc.block.find(sc.src, i+2, blockEnd)+1, len(sc.src))
}

// find returns the position of the first sep in src[from:], or len(src) if there is none.
// A search from any position in [c.from, c.end] reuses the cached result.
func (c *commentEnd) find(src []byte, from int, sep []byte) int {
	if from < c.from || from > c.end {
		c.from, c.end = from, len(src)
		if k := bytes.Index(src[from:], sep); k >= 0 {
			c.end = from + k
		}
	}
	return c.end
}
