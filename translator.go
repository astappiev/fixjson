package fixjson

func translate(src []byte) []byte {
	var (
		escaped            bool
		quote              bool
		inHTMLQuotedString bool
		inEmbeddedQuote    bool
		requireNewLine     bool
		stack              []byte // to track { and [
	)

	dst := make([]byte, 0, len(src)+16)
	for i := 0; i < len(src); i++ {
		ch := src[i]

		// always add escape character and escaped character
		if ch == '\\' || escaped {
			if !escaped && quote { // start of escape sequence in quotes
				if i+1 < len(src) && src[i+1] == '\'' {
					continue // skip escape character
				}
			}

			dst = append(dst, ch)
			escaped = !escaped
			continue
		}

		// find html escaped quote character (&quot;)
		if (!quote || inHTMLQuotedString) && hasPrefixAt(src, i, "&quot;") {
			ch = '"' // replace with quote
			i += 5   // go to last char or sequence, len("&quot;") - 1
			inHTMLQuotedString = !inHTMLQuotedString
		}

		// start or end of string literal
		if ch == '"' {
			if quote { // possible end of string literal
				if inEmbeddedQuote { // end of escaping
					next, _, _ := nextNonSpace(src, i+1)
					if expectedAfterQuote(next) {
						inEmbeddedQuote = false
						inHTMLQuotedString = false
						quote = false
						dst = append(dst, ch)
					} else {
						dst = append(dst, '\\', '"')
						inEmbeddedQuote = false
						continue
					}
				} else {
					next, spaces, _ := nextNonSpace(src, i+1)
					if !expectedAfterQuote(next) {
						if spaces == 0 { // if no spaces between quote and text, probably we need to escape it
							dst = append(dst, '\\', '"')
							inEmbeddedQuote = true
							continue
						}
					}
					quote = false
					inHTMLQuotedString = false
					dst = append(dst, ch)
				}
			} else {
				quote = true
				dst = append(dst, ch)
			}

			if !quote { // end of string literal
				next, _, _ := nextNonSpace(src, i+1)
				if next == '"' { // check if comma is missing
					dst = append(dst, ',')
				}

				if requireNewLine { // add new line that was previously skipped
					if i+1 < len(src) && src[i+1] == ',' { // after comma, if comma follows string literal
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
			switch ch {
			case '\t': // escape tabs
				dst = append(dst, '\\', 't')
			case '\r': // escape returns
				dst = append(dst, '\\', 'r')
			case '\n': // escape new lines
				dst = append(dst, '\\', 'n')
				requireNewLine = true
			default:
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
		if ch == '#' || (ch == '/' && i+1 < len(src) && src[i+1] == '/') {
			i = skipLineComment(src, i)
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
		if ch == '/' && i+1 < len(src) && src[i+1] == '*' {
			i = skipBlockComment(src, i)
			continue
		}

		// remove trailing commas
		if ch == ',' {
			next, _, _ := nextNonSpace(src, i+1)
			if next == '}' || next == ']' {
				continue // skip comma
			}
		}

		// seems to be a normal character, add it to the output
		dst = append(dst, ch)
	}

	if quote {
		dst = append(dst, '"')
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
	return ch == ',' || ch == ':' || ch == ']' || ch == '}' || ch == 0
}

func hasPrefixAt(src []byte, i int, s string) bool {
	return i >= 0 && len(src)-i >= len(s) && string(src[i:i+len(s)]) == s
}

// nextNonSpace finds the next character that is not a whitespace or comment.
// It returns the character, the number of whitespace characters skipped, and true if a character was found.
// Note: It treats EOF as found=false, ch=0
func nextNonSpace(src []byte, begin int) (ch byte, skipped int, found bool) {
	skipped = 0
	for i := begin; i < len(src); {
		if src[i] <= ' ' {
			skipped++
			i++
			continue
		}
		if src[i] == '#' || (src[i] == '/' && i+1 < len(src) && src[i+1] == '/') {
			start := i
			i = skipLineComment(src, i)
			skipped += i - start
			continue
		}
		if src[i] == '/' && i+1 < len(src) && src[i+1] == '*' {
			start := i
			i = skipBlockComment(src, i)
			skipped += i - start + 1
			i++
			continue
		}
		return src[i], skipped, true
	}
	return 0, skipped, false
}

func skipLineComment(src []byte, i int) int {
	for i++; i < len(src); i++ {
		if src[i] == '\n' {
			return i
		}
	}
	return len(src)
}

func skipBlockComment(src []byte, i int) int {
	for i += 2; i < len(src)-1; i++ {
		if src[i] == '*' && src[i+1] == '/' {
			return i + 1
		}
	}
	return len(src)
}
