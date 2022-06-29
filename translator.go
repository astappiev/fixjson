package fixjson

const (
	TAB          = 9   // \t
	NEWLINE      = 10  // \n
	RETURN       = 13  // \r
	SPACE        = 32  //
	QUOTE        = 34  // "
	HASH         = 35  // #
	APOSTROPHE   = 39  // "
	AMPERSAND    = 38  // &
	ASTERISK     = 42  // *
	COMMA        = 44  // ,
	SLASH        = 47  // /
	COLON        = 58  // :
	ESCAPE       = 92  // \
	LEFT_SQUARE  = 91  // [
	RIGHT_SQUARE = 93  // ]
	LEFT_CURLY   = 123 // {
	RIGHT_CURLY  = 125 // }
)

func translate(src []byte) []byte {
	var (
		escaped        bool
		quote          bool
		escapedQuote   bool
		escapingQuote  bool
		requireNewLine bool
	)

	var dst []byte
	for i := 0; i < len(src); i++ {
		ch := src[i]

		// always add escape character and escaped character
		if ch == ESCAPE || escaped {
			if !escaped && quote { // start of escape sequence in quotes
				if eq(src, i+1, APOSTROPHE) {
					continue // skip escape character
				}
			}

			dst = append(dst, ch)
			escaped = !escaped
			continue
		}

		// find html escaped quote character (&quot;)
		if (!quote || quote && escapedQuote) && seq(src, i, "&quot;") {
			ch = QUOTE // replace with quote
			i += 5     // go to last char or sequence, len("&quot;") - 1
			escapedQuote = !escapedQuote
		}

		// start or end of string literal
		if ch == QUOTE {
			if quote { // possible end of string literal
				if escapingQuote { // end of escaping
					dst = append(dst, '\\', '"')
					escapingQuote = false
					continue
				}

				fch, spaces := firstNotSpace(src, i+1)
				if !expectedAfterQuote(fch) {
					if spaces == 0 { // if no spaces between quote and text, probably we need to escape it
						dst = append(dst, '\\', '"')
						escapingQuote = true
						continue
					}
				}
			}

			quote = !quote
			dst = append(dst, ch)

			if !quote { // end of string literal
				fch, _ := firstNotSpace(src, i+1)
				if fch == QUOTE { // check if comma is missing
					dst = append(dst, COMMA)
				}

				if requireNewLine { // add new line that was previously skipped
					if eq(src, i+1, COMMA) { // after comma, if comma follows string literal
						dst = append(dst, ',')
						i++
					}
					dst = append(dst, '\n')
					requireNewLine = false
				}
			}
			continue
		}

		// inside string literal
		if quote {
			if src[i] == TAB { // escape tabs
				dst = append(dst, '\\', 't')
				continue
			}

			if src[i] == RETURN { // escape returns
				dst = append(dst, '\\', 'r')
				continue
			}

			if ch == NEWLINE { // escape new lines
				dst = append(dst, '\\', 'n')
				requireNewLine = true
				continue
			}

			dst = append(dst, ch)
			continue
		}

		// start of inline comment
		if ch == HASH || (ch == SLASH && eq(src, i+1, SLASH)) {
			for i++; i < len(src); i++ { // skip all characters until newline
				if src[i] == NEWLINE {
					if src[i-1] == RETURN { // add carriage return if needed
						dst = append(dst, RETURN, NEWLINE)
					} else {
						dst = append(dst, NEWLINE)
					}
					break
				}
			}
			continue
		}

		// start of multi-line comment
		if ch == SLASH && eq(src, i+1, ASTERISK) {
			for i += 2; i < len(src)-1; i++ { // skip all characters until end of comment
				if src[i] == ASTERISK && src[i+1] == SLASH { // end of multi-line comment
					i++
					break
				}
			}
			continue
		}

		// remove trailing commas
		if ch == COMMA {
			fch, _ := firstNotSpace(src, i+1)
			if fch == RIGHT_CURLY || fch == RIGHT_SQUARE {
				continue // skip comma
			}
		}

		// seems to be a normal character, add it to the output
		dst = append(dst, ch)
	}
	return dst
}

func expectedAfterQuote(ch byte) bool {
	return ch == COMMA || ch == COLON || ch == RIGHT_SQUARE || ch == RIGHT_CURLY
}

func seq(src []byte, i int, s string) bool {
	if len(src) >= i+len(s) {
		for j := 0; j < len(s); j++ {
			if src[i+j] != s[j] {
				return false
			}
		}
		return true
	}
	return false
}

func eq(src []byte, i int, ch byte) bool {
	return len(src) >= i && src[i] == ch
}

func firstNotSpace(src []byte, begin int) (byte, int) {
	for i := begin; i < len(src); i++ {
		if src[i] > SPACE {
			return src[i], i - begin
		}
	}
	return 0, -1
}
