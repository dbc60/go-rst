// go-rst - A reStructuredText parser for Go
// 2014 (c) The go-rst Authors
// MIT Licensed. See LICENSE for details.

package parse

import (
	"fmt"
	"github.com/demizer/go-elog"
)

type systemMessageLevel int

const (
	levelInfo systemMessageLevel = iota
	levelWarning
	levelError
	levelSevere
)

var systemMessageLevels = [...]string{
	"INFO",
	"WARNING",
	"ERROR",
	"SEVERE",
}

func (s systemMessageLevel) String() string { return systemMessageLevels[s] }

type systemMessage struct {
	level  systemMessageLevel
	line   int
	source string
	items  []item
}

type sectionLevel struct {
	char rune // The adornment character used to describe the section
	overline bool // The section contains an overline
	length int // The length of the adornment lines
}

type sectionLevels []sectionLevel

func (s *sectionLevels) String() string {
	var out string
	for lvl, sec := range *s {
		out += fmt.Sprintf("level: %d, rune: %q, overline: %t, length: %d\n",
			lvl+1, sec.char, sec.overline, sec.length)
	}
	return out
}

func (s *sectionLevels) Add(adornChar rune, overline bool, length int) int {
	lvl := s.Find(adornChar)
	if lvl > 0 {
		return lvl
	}
	*s = append(*s, sectionLevel{char: adornChar, overline: overline, length: length})
	return len(*s)
}

// Returns -1 if not found
func (s *sectionLevels) Find(adornChar rune) int {
	for lvl, sec := range *s {
		if sec.char == adornChar {
			return lvl+1
		}
	}
	return -1
}

func (s *sectionLevels) Level() int {
	return len(*s)
}

func Parse(name, text string) (t *Tree, err error) {
	t = New(name)
	t.text = text
	_, err = t.Parse(text, t)
	return
}

func New(name string) *Tree {
	return &Tree{Name: name, sectionLevels: new(sectionLevels)}
}

type Tree struct {
	Name      string
	text      string
	Root      *ListNode
	lex       *lexer
	peekCount int
	token     [3]item // three-token look-ahead for parser.
	sectionLevel int // The current section level of parsing
	sectionLevels *sectionLevels // Encountered section levels
}

// startParse initializes the parser, using the lexer.
func (t *Tree) startParse(lex *lexer) {
	t.Root = nil
	t.lex = lex
}

// stopParse terminates parsing.
func (t *Tree) stopParse() {
	t.lex = nil
}

func (t *Tree) Parse(text string, treeSet *Tree) (tree *Tree, err error) {
	log.Debugln("Start")
	t.startParse(lex(t.Name, text))
	t.text = text
	t.parse(treeSet)
	log.Debugln("End")
	return t, nil
}

func (t *Tree) parse(tree *Tree) (next Node) {
	log.Debugln("Start")
	t.Root = newList(t.peek().Position)
	for t.peek().ElementType != itemEOF {
		switch token := t.next(); token.ElementType {
		case itemBlankLine:
			log.Debugln("Found itemBlankLine")
		case itemSectionAdornment:
			log.Debugln("Found itemSectionAdornment")
		case itemTitle:
			log.Debugln("Found itemTitle")
		case itemParagraph:
			log.Debugln("Found itemParagraph")
		}
	}
	log.Debugln("End")
	return nil
}

// peek returns but does not consume the next token.
func (t *Tree) peek() item {
	if t.peekCount > 0 {
		return t.token[t.peekCount-1]

	}
	t.peekCount = 1
	t.token[0] = t.lex.nextItem()
	return t.token[0]
}
