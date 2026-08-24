// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-pgx Authors
// SPDX-License-Identifier: Apache-2.0

package queries

import "strings"

// stripComments removes SQL comments and collapses the whitespace around them,
// so the server receives only statement text.
//
// Quoted content is preserved, for example inside a literal: `--`. The scanner
// tracks single-quoted strings, escape strings (E'...', where a backslash
// escapes), double-quoted identifiers and dollar quoting. Block comments nest,
// so /* /* */ */ is one comment.
func stripComments(statement string) string {
	var out strings.Builder
	out.Grow(len(statement))

	// Runs of whitespace and comments collapse to a single separator, written
	// only when something follows them, so the result has no padding at either
	// end. That separator is a space, except between two adjacent string
	// literals. Postgres joins those into one string only when a newline
	// separates them, so putting them on one line would be a syntax error. The
	// newline is kept only where the source already had one, which leaves a
	// broken query broken.
	spacePending, newlinePending, blockCommentPending := false, false, false
	lastByte := byte(0)
	emit := func(text string) {
		if spacePending && out.Len() > 0 {
			if newlinePending && lastByte == '\'' && text[0] == '\'' {
				out.WriteByte('\n')
			} else {
				out.WriteByte(' ')
			}
		}
		spacePending, newlinePending, blockCommentPending = false, false, false
		out.WriteString(text)
		lastByte = text[len(text)-1]
	}

	for index := 0; index < len(statement); {
		char := statement[index]

		switch {
		// scan.l counts a `--` comment as whitespace in the rule that joins two
		// string constants, and does not admit block comments at all. So a line
		// comment leaves the parts joined and a block comment separates them,
		// regardless of how many newlines surround it.
		case char == '-' && index+1 < len(statement) && statement[index+1] == '-':
			for index < len(statement) && statement[index] != '\n' {
				index++
			}
			spacePending, newlinePending = true, false

		case char == '/' && index+1 < len(statement) && statement[index+1] == '*':
			index = skipBlockComment(statement, index)
			spacePending, newlinePending, blockCommentPending = true, false, true

		case isSQLSpace(char):
			for index < len(statement) && isSQLSpace(statement[index]) {
				if statement[index] == '\n' && !blockCommentPending {
					newlinePending = true
				}
				index++
			}
			spacePending = true

		case char == '\'' || char == '"':
			end := scanQuoted(statement, index, char, honoursBackslash(statement, index))
			emit(statement[index:end])
			index = end

		case char == '$' && !precededByIdentifier(statement, index):
			if end, isDollarQuote := scanDollarQuoted(statement, index); isDollarQuote {
				emit(statement[index:end])
				index = end
				continue
			}
			emit(statement[index : index+1])
			index++

		default:
			emit(statement[index : index+1])
			index++
		}
	}
	return out.String()
}

// skipBlockComment returns the index just past the block comment at start,
// counting nesting depth. An unterminated comment consumes the rest of the
// statement.
func skipBlockComment(statement string, start int) int {
	depth := 0
	index := start
	for index < len(statement) {
		if statement[index] == '/' && index+1 < len(statement) && statement[index+1] == '*' {
			depth++
			index += 2
			continue
		}
		if statement[index] == '*' && index+1 < len(statement) && statement[index+1] == '/' {
			depth--
			index += 2
			if depth == 0 {
				return index
			}
			continue
		}
		index++
	}
	return index
}

// scanQuoted returns the index just past the quoted run at start. A doubled
// delimiter escapes rather than closes. With backslashEscapes, true only of an
// E'...' string, a backslash escapes the next character.
func scanQuoted(statement string, start int, quote byte, backslashEscapes bool) int {
	index := start + 1
	for index < len(statement) {
		char := statement[index]
		if backslashEscapes && char == '\\' && index+1 < len(statement) {
			index += 2
			continue
		}
		if char == quote {
			if index+1 < len(statement) && statement[index+1] == quote {
				index += 2
				continue
			}
			return index + 1
		}
		index++
	}
	return index
}

// honoursBackslash reports whether the string at quoteIndex is an escape
// string, E'...'. Only there does a backslash escape. Under the default
// standard_conforming_strings a backslash is an ordinary byte, and escaping it
// would swallow the closing quote and mis-read the rest.
func honoursBackslash(statement string, quoteIndex int) bool {
	if statement[quoteIndex] != '\'' || quoteIndex == 0 {
		return false
	}
	prefix := statement[quoteIndex-1]
	if prefix != 'E' && prefix != 'e' {
		return false
	}
	// An E ending an identifier, as in `value'...'`, is not a prefix.
	return quoteIndex < 2 || !isIdentifierByte(statement[quoteIndex-2])
}

// precededByIdentifier reports whether a $ at index can open a dollar quote.
// Postgres requires whitespace before one, so `a$b$` is the single identifier
// a$b$; without this check the scanner reads $b$ as an opening delimiter and
// swallows the rest of the statement.
func precededByIdentifier(statement string, index int) bool {
	return index > 0 && isIdentifierByte(statement[index-1])
}

// scanDollarQuoted returns the index just past a dollar-quoted run, and whether
// one was found. A placeholder such as $1 is not a dollar quote.
func scanDollarQuoted(statement string, start int) (int, bool) {
	tagEnd := start + 1
	for tagEnd < len(statement) && isIdentifierByte(statement[tagEnd]) {
		// A tag cannot begin with a digit.
		if tagEnd == start+1 && statement[tagEnd] >= '0' && statement[tagEnd] <= '9' {
			return start, false
		}
		tagEnd++
	}
	if tagEnd >= len(statement) || statement[tagEnd] != '$' {
		return start, false
	}
	delimiter := statement[start : tagEnd+1]
	closing := strings.Index(statement[tagEnd+1:], delimiter)
	if closing < 0 {
		return len(statement), true
	}
	return tagEnd + 1 + closing + len(delimiter), true
}

func isSQLSpace(char byte) bool {
	return char == ' ' || char == '\t' || char == '\n' || char == '\r' || char == '\f' || char == '\v'
}

func isIdentifierByte(char byte) bool {
	return char == '_' ||
		(char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') ||
		char >= 0x80 // a multi-byte identifier character
}
