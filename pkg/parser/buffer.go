package parser

import (
	tok "github.com/demizer/go-rst/pkg/token"
)

var (
	initialCapacity = 100
)

type tokenBuffer struct {
	token *tok.Item
	buf   []*tok.Item
	lex   *tok.Lexer
	index int
}

func newTokenBuffer(l *tok.Lexer) *tokenBuffer {
	return &tokenBuffer{
		buf: make([]*tok.Item, initialCapacity),
		lex: l,
	}
}

func (t *tokenBuffer) append(item *tok.Item) {
	t.buf = append(t.buf, item)
	t.index = len(t.buf) - 1
	t.token = t.buf(t.index)
}

// backup shifts the token buf right one position.
func (t *tokenBuffer) backup() (tok *tok.Item) {
	if t.index > 0 {
		t.index--
	}
	tok = t.buf[t.index]
	p.Msgr("buffer index item", "index", t.index, "token", t.buf[t.index])
	return
}

// peekBack uses the token buf to "look back" a number of positions (pos). Looking back more positions than the
// Parser.token buf allows (3) will generate a panic.
func (t *tokenBuffer) peekBack(pos int) (tok *tok.Item) {
	if t.index-pos > 0 {
		tok = t.buf[t.index-pos]
	}
	return
}

func (t *tokenBuffer) peekBackTo(item tok.Type) (tok *tok.Item) {
	for i := t.index - 1; i >= 0; i-- {
		if t.buf[i] != nil && t.buf[i].Type == item {
			if i >= 0 {
				tok = t.buf[i]
			}
			break
		}
	}
	return
}

// peek looks ahead in the token stream a number of positions (pos) and gets the next token from the lexer. A pointer to the
// token is kept in the Parser.token buf. If a token pointer already exists in the buf, that token is used instead
// and no buf are received the the lexer stream (channel).
func (t *tokenBuffer) peek(pos int) *tok.Item {
	nItem := t.buf[t.index]
	for i := 1; i <= pos; i++ {
		if t.buf[t.index+i] != nil {
			nItem = t.buf[t.index+i]
			continue
		} else {
			if t.lex == nil {
				continue
			}
			// p.Msg("Getting next item")
			t.buf[t.index+i] = t.lex.NextItem()
			nItem = t.buf[t.index+i]
		}
	}
	p.Msgr("peek token", "index", t.index, "token", nItem)
	return nItem
}

// peekSkip looks ahead one position skipiing a specified itemElement. If that element is found, a pointer is returned,
// otherwise nil is returned.
func (t *tokenBuffer) peekSkip(iSkip tok.Type) *tok.Item {
	var nItem *tok.Item
	count := 1
	for {
		nItem = p.peek(count)
		if nItem.Type != iSkip {
			break
		}
		count++
	}
	return nItem
}

func (t *tokenBuffer) next(pos int) *tok.Item {
	if pos == 0 {
		return t.buf[t.index]
	}
	t.index++
	if t.buf[t.index] == nil && t.lex != nil {
		t.buf[t.index] = t.lex.NextItem()
	}
	pos--
	if pos > 0 {
		t.next(pos)
	}
	return t.buf[t.index]
}

// reset clears the token buf
func (t *tokenBuffer) reset(begin, end int) {
	t.index = 0
	t.buf = nil
	t.buf = make([]*tok.Item, initialCapacity)
}
