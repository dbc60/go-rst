// go-rst - A reStructuredText parser for Go
// 2014 (c) The go-rst Authors
// MIT Licensed. See LICENSE for details.

// Package parse is a reStructuredText parser implemented in Go!
//
// This package is only meant for lexing and parsing reStructuredText. See the
// top level package documentation for details on using the go-rst package
// package API.
package parse

import (
	"reflect"

	"code.google.com/p/go.text/unicode/norm"
	"github.com/demizer/go-elog"
	"github.com/demizer/go-spew/spew"
)

// Used for debugging only
var spd = spew.ConfigState{Indent: "\t", DisableMethods: true}

// systemMessageLevel implements four levels for messages and is used in
// conjunction with the parserMessage type.
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

// String implments Stringer and return a string of the systemMessageLevel.
func (s systemMessageLevel) String() string {
	return systemMessageLevels[s]
}

// parserMessage implements messages generated by the parser. Parser messages
// are leveled using systemMessageLevels.
type parserMessage int

const (
	warningShortOverline parserMessage = iota
	warningShortUnderline
	errorInvalidSectionOrTransitionMarker
	severeUnexpectedSectionTitle
	severeUnexpectedSectionTitleOrTransition
	severeIncompleteSectionTitle
	severeMissingMatchingUnderlineForOverline
	severeOverlineUnderlineMismatch
)

var parserErrors = [...]string{
	"warningShortOverline",
	"warningShortUnderline",
	"errorInvalidSectionOrTransitionMarker",
	"severeUnexpectedSectionTitle",
	"severeUnexpectedSectionTitleOrTransition",
	"severeIncompleteSectionTitle",
	"severeMissingMatchingUnderlineForOverline",
	"severeOverlineUnderlineMismatch",
}

// String implements Stringer and returns the parserMessage as a string. The
// returned string is the parserMessage name, not the message itself.
func (p parserMessage) String() string {
	return parserErrors[p]
}

// Message returns the message of the parserMessage as a string.
func (p parserMessage) Message() (s string) {
	switch p {
	case warningShortOverline:
		s = "Title overline too short."
	case warningShortUnderline:
		s = "Title underline too short."
	case errorInvalidSectionOrTransitionMarker:
		s = "Invalid section title or transition marker."
	case severeUnexpectedSectionTitle:
		s = "Unexpected section title."
	case severeUnexpectedSectionTitleOrTransition:
		s = "Unexpected section title or transition."
	case severeIncompleteSectionTitle:
		s = "Incomplete section title."
	case severeMissingMatchingUnderlineForOverline:
		s = "Missing matching underline for section title overline."
	case severeOverlineUnderlineMismatch:
		s = "Title overline & underline mismatch."
	}
	return
}

// Level returns the parserMessage level.
func (p parserMessage) Level() (s systemMessageLevel) {
	lvl := int(p)
	switch {
	case lvl <= 1:
		s = levelWarning
	case lvl == 2:
		s = levelError
	case lvl >= 3:
		s = levelSevere
	}
	return
}

// sectionLevels contains the encountered sections as pointers to a
// SectionNode.
type sectionLevels []*SectionNode

// FindByRune loops through the sectionLevels to find a section using a Rune as
// the key. If the section is found, a pointer to the SectionNode is returned.
func (s *sectionLevels) FindByRune(adornChar rune) *SectionNode {
	for _, sec := range *s {
		if sec.UnderLine.Rune == adornChar {
			return sec
		}
	}
	return nil
}

// If exists == true, a section node with the same text and underline has been found in
// sectionLevels, sec is the matching SectionNode. If exists == false, then the sec return value is
// the similarly leveled SectionNode. If exists == false and sec == nil, then the SectionNode added
// to sectionLevels is a new Node.
func (s *sectionLevels) Add(section *SectionNode) (exists bool, sec *SectionNode) {
	sec = s.FindByRune(section.UnderLine.Rune)
	if sec != nil {
		if sec.Title.Text != section.Title.Text {
			section.Level = sec.Level
		}
	} else {
		section.Level = len(*s) + 1
	}
	exists = false
	*s = append(*s, section)
	return
}

// Level returns the Level as an integer.
func (s *sectionLevels) Level() int {
	return len(*s)
}

// Parse is the entry point for the reStructuredText parser. Errors generated
// by the parser are returned as a NodeList.
func Parse(name, text string) (t *Tree, errors *NodeList) {
	t = New(name, text)
	if !norm.NFC.IsNormalString(text) {
		text = norm.NFC.String(text)
	}
	t.Parse(text, t)
	errors = t.Messages
	return
}

// New returns a fresh parser tree.
func New(name, text string) *Tree {
	return &Tree{
		Name:          name,
		Nodes:         newList(),
		Messages:      newList(),
		text:          text,
		nodeTarget:    newList(),
		sectionLevels: new(sectionLevels),
		indentWidth:   indentWidth,
	}
}

const (
	// The middle of the Tree.token buffer so that there are three possible
	// "backup" token positions and three forward "peek" positions.
	zed = 3

	// Default indent width
	indentWidth = 4
)

// Tree contains the parser tree. The Nodes field contains the parsed nodes of
// the input input data.
type Tree struct {
	Name          string    // The name of the current parser input
	Nodes         *NodeList // The root node list
	Messages      *NodeList // Messages generated by the parser
	nodeTarget    *NodeList // Used by the parser to add nodes to a target NodeList
	text          string    // The input text
	lex           *lexer
	token         [7]*item
	sectionLevels *sectionLevels // Encountered section levels
	id            int            // The consecutive id of the node in the tree
	indentWidth   int
	indentLevel   int
}

// startParse initializes the parser, using the lexer.
func (t *Tree) startParse(lex *lexer) {
	t.lex = lex
}

// Parse activates the parser using text as input data. A parse tree is
// returned on success or failure. Users of the Parse package should use the
// Top level Parse function.
func (t *Tree) Parse(text string, treeSet *Tree) (tree *Tree) {
	log.Debugln("Start")
	t.startParse(lex(t.Name, text))
	t.text = text
	t.parse(treeSet)
	log.Debugln("End")
	return t
}

// parse is where items are retrieved from the parser and dispatched according
// to the itemElement type.
func (t *Tree) parse(tree *Tree) {
	log.Debugln("Start")

	t.nodeTarget = t.Nodes

	for t.peek(1).Type != itemEOF {
		var n Node

		token := t.next()
		log.Infof("\nParser got token: %#+v\n\n", token)

		switch token.Type {
		case itemSectionAdornment:
			n = t.section(token)
		case itemParagraph:
			n = newParagraph(token, &t.id)
		case itemSpace:
			n = t.indent(token)
			if n == nil {
				continue
			}
		case itemTitle, itemBlankLine:
			// itemTitle is consumed when evaluating itemSectionAdornment
			continue
		case itemTransition:
			n = newTransition(token, &t.id)
		}

		t.nodeTarget.append(n)
		switch n.NodeType() {
		case NodeSection, NodeBlockQuote:
			// Set the loop to append items to the NodeList of the new section
			t.nodeTarget = reflect.ValueOf(n).Elem().FieldByName("NodeList").Addr().Interface().(*NodeList)
		}
	}

	log.Debugln("End")
}

// peekBack uses the token buffer to "look back" a number of positions (pos).
// Looking back more positions than the Tree.token buffer allows (3) will
// generate a panic.
func (t *Tree) peekBack(pos int) *item {
	return t.token[zed-pos]
}

// peek looks ahead in the token stream a number of positions (pos) and gets
// the next token from the lexer. A pointer to the token is kept in the
// Tree.token buffer. If a token pointer already exists in the buffer, that
// token is used instead and no tokens are received the the lexer stream
// (channel).
func (t *Tree) peek(pos int) *item {
	// log.Debugln("\n", "Pos:", pos)
	// log.Debugf("##### peek() before #####\n")
	// spd.Dump(t.token)
	nItem := t.token[zed]
	for i := 1; i <= pos; i++ {
		if t.token[zed+i] != nil {
			nItem = t.token[zed+i]
			log.Debugf("Using %#+v\n", nItem)
			continue
		} else {
			log.Debugln("Getting next item")
			t.token[zed+i] = t.lex.nextItem()
			nItem = t.token[zed+i]
		}
	}
	// log.Debugf("\n##### peek() aftermath #####\n")
	// spd.Dump(t.token)
	// log.Debugf("Returning: %#+v\n", nItem)
	return nItem
}

// peekSkip looks ahead one position skipiing a specified itemElement. If that
// element is found, a pointer is returned, otherwise nil is returned.
func (t *Tree) peekSkip(iSkip itemElement) *item {
	var nItem *item
	count := 1
	for {
		nItem = t.peek(count)
		if nItem.Type != iSkip {
			break
		}
		count++
	}
	return nItem
}

// next is the workhorse of the parser. It is repsonsible for getting the next
// token from the lexer stream (channel). If the next token already exists in
// the token buffer, than the token buffer is shifted left and the pointer to
// the "zed" token is returned.
func (t *Tree) next() *item {
	// log.Debugf("\n##### next() before #####\n")
	// spd.Dump(t.token)
	for x := 0; x < len(t.token)-1; x++ {
		t.token[x] = t.token[x+1]
		t.token[x+1] = nil
	}
	if t.token[zed] == nil && t.lex != nil {
		t.token[zed] = t.lex.nextItem()
	}
	// log.Debugf("\n##### next() aftermath #####\n\n")
	// spd.Dump(t.token)
	return t.token[zed]
}

// section is responsible for parsing the title, overline, and underline tokens
// returned from the parser. If there are errors parsing these elements, than a
// systemMessage is generated and added to Tree.Nodes.
func (t *Tree) section(i *item) Node {
	log.Debugln("Start")
	var overAdorn, indent, title, underAdorn *item

	if pBack := t.peekBack(1); pBack != nil && pBack.Type == itemTitle {
		// Section with no overline
		title = t.peekBack(1)
		underAdorn = i
	} else if pBack := t.peekBack(1); pBack != nil && pBack.Type == itemSpace {
		// Indented section (error)
		if t.peekBack(2).Type == itemTitle {
			return t.systemMessage(severeUnexpectedSectionTitle)
		}
		return t.systemMessage(severeUnexpectedSectionTitleOrTransition)
	} else if pFor := t.peekSkip(itemSpace); pFor != nil && pFor.Type == itemTitle {
		// Section with overline
		overAdorn = i
		t.next()
	loop:
		for {
			switch tTok := t.token[zed]; tTok.Type {
			case itemTitle:
				title = tTok
				t.next()
			case itemSpace:
				indent = tTok
				t.next()
			case itemSectionAdornment:
				underAdorn = tTok
				break loop
			}
		}
	} else if pFor := t.peekSkip(itemSpace); pFor != nil && pFor.Type == itemParagraph {
		// If a section contains an itemParagraph, it is because the underline
		// is missing, therefore we generate an error based on what follows the
		// itemParagraph.
		t.next() // Move the token buffer past the error tokens
		t.next()
		if p := t.peek(1); p != nil && p.Type == itemBlankLine {
			return t.systemMessage(severeMissingMatchingUnderlineForOverline)
		}
		return t.systemMessage(severeIncompleteSectionTitle)
	} else if pFor := t.peekSkip(itemSpace); pFor != nil && pFor.Type == itemSectionAdornment {
		// Missing section title
		t.next() // Move the token buffer past the error token
		return t.systemMessage(errorInvalidSectionOrTransitionMarker)
	}

	if overAdorn != nil && overAdorn.Text.(string) != underAdorn.Text.(string) {
		return t.systemMessage(severeOverlineUnderlineMismatch)
	}

	sec := newSection(title, overAdorn, underAdorn, indent, &t.id)
	exists, eSec := t.sectionLevels.Add(sec)
	if !exists && eSec != nil {
		// There is a matching level in sectionLevels
		t.nodeTarget = &(*t.sectionLevels)[sec.Level-2].NodeList
	}

	oLen := title.Length
	if indent != nil {
		oLen = indent.Length + title.Length
	}
	if overAdorn != nil && oLen > overAdorn.Length {
		sec.NodeList = append(sec.NodeList, t.systemMessage(warningShortOverline))
	} else if overAdorn == nil && title.Length != underAdorn.Length {
		sec.NodeList = append(sec.NodeList, t.systemMessage(warningShortUnderline))
	}

	log.Debugln("End")
	return sec
}

// systemMessage generates a Node based on the passed parserMessage. The
// generated message is returned as a SystemMessageNode.
func (t *Tree) systemMessage(err parserMessage) Node {
	var lbText string
	var lbTextLen int
	var backToken int

	s := newSystemMessage(&item{
		Type: itemSystemMessage,
		Line: t.token[zed].Line,
	},
		err.Level(), &t.id)

	msg := newParagraph(&item{
		Text:   err.Message(),
		Length: len(err.Message()),
	}, &t.id)

	log.Debugln("FOUND", err)

	// spd.Dump(t.token)
	var overLine, indent, title, underLine, newLine string

	switch err {
	case errorInvalidSectionOrTransitionMarker:
		lbText = t.token[zed-1].Text.(string) + "\n" + t.token[zed].Text.(string)
		s.Line = t.token[zed-1].Line
		lbTextLen = len(lbText) + 1
	case warningShortOverline, severeOverlineUnderlineMismatch:
		backToken = zed - 2
		if t.peekBack(2).Type == itemSpace {
			backToken = zed - 3
			indent = t.token[zed-2].Text.(string)
		}
		overLine = t.token[backToken].Text.(string)
		title = t.token[zed-1].Text.(string)
		underLine = t.token[zed].Text.(string)
		newLine = "\n"
		lbText = overLine + newLine + indent + title + newLine + underLine
		s.Line = t.token[backToken].Line
		lbTextLen = len(lbText) + 2
	case warningShortUnderline, severeUnexpectedSectionTitle:
		backToken = zed - 1
		if t.peekBack(1).Type == itemSpace {
			backToken = zed - 2
		}
		lbText = t.token[backToken].Text.(string) + "\n" + t.token[zed].Text.(string)
		lbTextLen = len(lbText) + 1
	case severeIncompleteSectionTitle, severeMissingMatchingUnderlineForOverline:
		lbText = t.token[zed-2].Text.(string) + "\n" + t.token[zed-1].Text.(string) +
			t.token[zed].Text.(string)
		s.Line = t.token[zed-2].Line
		lbTextLen = len(lbText) + 1
	case severeUnexpectedSectionTitleOrTransition:
		lbText = t.token[zed].Text.(string)
		lbTextLen = len(lbText)
	}

	lb := newLiteralBlock(&item{
		Type:   itemLiteralBlock,
		Text:   lbText,
		Length: lbTextLen, // Add one to account for the backslash
	}, &t.id)

	s.NodeList = append(s.NodeList, msg, lb)
	t.Messages.append(s)

	return s
}

// indent parses IndentNode's returned from the lexer and returns a
// BlockQuoteNode.
func (t *Tree) indent(i *item) Node {
	level := i.Length / t.indentWidth
	if t.peekBack(1).Type == itemBlankLine {
		if t.indentLevel == level {
			// Append to the current blockquote NodeList
			return nil
		}
		t.indentLevel = level
		return newBlockQuote(&item{Type: itemBlockquote, Line: i.Line}, level, &t.id)
	}
	return nil
}
