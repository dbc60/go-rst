package token

import "unicode"

func isReferenceNameSimpleAllowedRune(r rune) bool {
	// allowed runes plus unicode categories
	allowedRunes := [...]rune{'_', '-', '+', '.', ':'}
	allowedCats := []*unicode.RangeTable{unicode.Nd, unicode.Ll, unicode.Lt}
	for _, a := range allowedRunes {
		if a == r {
			return true
		} else if unicode.In(r, allowedCats...) {
			return true
		}
	}
	return false
}

func isReferenceNameSimple(l *Lexer, fromPos int) bool {
	count := fromPos
	for {
		p := l.peek(count)
		if p == ':' {
			break
		} else if unicode.IsSpace(p) {
			l.Msg("NOT FOUND")
			return false
		} else if p == EOL {
			l.Msg("NOT FOUND")
			return false
		} else if !isReferenceNameSimpleAllowedRune(p) {
			l.Msg("NOT FOUND")
			return false
		}
		count++
	}
	l.Msg("FOUND")
	return true
}

func isReferenceNamePhrase(l *Lexer, fromPos int) bool {
	count := fromPos
	words := 0
	openTick := false
	for {
		p := l.peek(count)
		if p == EOL && fromPos == count {
			// At end of line, so ref is not possible
			l.Msg("NOT FOUND")
			return false
		}
		if p == '`' {
			if openTick && l.peek(count+1) == ':' {
				break
			}
			openTick = true
		} else if p == EOL {
			if words == 0 {
				l.Msg("NOT FOUND")
				return false
			}
			break
		} else if unicode.IsSpace(p) {
			words++
		}
		count++
	}
	l.Msg("FOUND")
	return true
}
