// Package parse is a reStructuredText parser implemented in Go!
//
// This package is only meant for lexing and parsing reStructuredText. See the top level package documentation for details on
// using the go-rst package package API.
package parse

import (
	"errors"
	"fmt"
	"os"

	"github.com/davecgh/go-spew/spew"
	"golang.org/x/text/unicode/norm"
)

// Used for debugging only
var spd = spew.ConfigState{ContinueOnMethod: true, Indent: "\t", MaxDepth: 0} //, DisableMethods: true}

const (
	// The middle of the Tree.token buffer so that there are three possible "backup" token positions and three forward
	// "peek" positions.
	zed = 4

	// Default indent width
	indentWidth = 4
)

// Tree contains the parser tree. The Nodes field contains the parsed nodes of the input input data.
type Tree struct {
	Name     string   // The name of the current parser input
	Nodes    NodeList // The root node list
	Messages NodeList // Messages generated by the parser

	nodeTarget *NodeList    // Used to append nodes to a target NodeList
	text       string       // The input text
	lex        *lexer       // The place where tokens come from
	token      [9]*item     // Token buffer, number 4 is the middle. AKA the "zed" token
	indents    *indentQueue // Indent level tracking

	sectionLevels *sectionLevels // Encountered section levels
	sections      []*SectionNode // Pointers to encountered sections

	openList Node // Open Bullet List, Enum List, or Definition List

	logCtx *logCtx // Use this to log to the "parse" context, mainly for debugging.
}

// New returns a fresh parser tree.
func New(name, text string) *Tree {
	t := &Tree{
		Name:          name,
		text:          text,
		sectionLevels: new(sectionLevels),
		indents:       new(indentQueue),
		logCtx:        NewLogCtx("parser"),
	}
	t.nodeTarget = &t.Nodes
	return t
}

// Parse is the entry point for the reStructuredText parser. Errors generated by the parser are returned as a NodeList.
func Parse(name, text string) (t *Tree, errors NodeList) {
	t = New(name, text)
	if !norm.NFC.IsNormalString(text) {
		text = norm.NFC.String(text)
	}
	t.Parse(text, t)
	errors = t.Messages
	return
}

func (t *Tree) log(keyvals ...interface{}) { t.logCtx.Log(keyvals...) }

func (t *Tree) logMsg(message string) { t.logCtx.Log("msg", message) }

// startParse initializes the parser, using the lexer.
func (t *Tree) startParse(lex *lexer) {
	t.lex = lex
}

// Parse activates the parser using text as input data. A parse tree is returned on success or failure. Users of the Parse
// package should use the Top level Parse function.
func (t *Tree) Parse(text string, treeSet *Tree) (tree *Tree) {
	t.startParse(lex(t.Name, []byte(text)))
	t.text = text
	t.parse(treeSet)
	return t
}

// parse is where items are retrieved from the parser and dispatched according to the itemElement type.
func (t *Tree) parse(tree *Tree) {
	for {
		var n interface{}

		token := t.next(1)
		if token == nil || token.Type == itemEOF {
			break
		}

		t.log("msg", "Parser got token", "token", token)

		switch token.Type {
		case itemText:
			t.paragraph(token)
		case itemInlineEmphasisOpen:
			t.inlineEmphasis(token)
		case itemInlineStrongOpen:
			t.inlineStrong(token)
		case itemInlineLiteralOpen:
			t.inlineLiteral(token)
		case itemInlineInterpretedTextOpen:
			t.inlineInterpretedText(token)
		case itemInlineInterpretedTextRoleOpen:
			t.inlineInterpretedTextRole(token)
		case itemTransition:
			newTransition(token)
		case itemCommentMark:
			t.comment(token)
		case itemSectionAdornment:
			t.section(token)
		case itemEnumListArabic:
			n = t.enumList(token)
			// FIXME: This is only until enumerated list are properly implemented.
			if n == nil {
				continue
			}
		case itemSpace:
			// //
			// //  FIXME: Blockquote parsing is NOT fully implemented.
			// //
			// if t.peekBack(1).Type == itemBlankLine && t.indentLevel == 0 {
			// t.blockquote(token)
			// }
			// if n == nil {
			// // The calculated indent level was the same as the current indent level.
			// continue
			// }
		case itemBlankLine, itemTitle, itemEscape:
			// itemTitle is consumed when evaluating itemSectionAdornment
			continue
		case itemBlockQuote:
			t.blockquote(token)
		case itemDefinitionTerm:
			t.definitionTerm(token)
		case itemBullet:
			t.bulletList(token)
		default:
			t.log(fmt.Errorf("Token type: %q is not yet supported in the parser", token.Type.String()))
		}
	}
}

func (t *Tree) subParseBodyElements(token *item) Node {
	t.log("msg", "Have token", "tokenType", token.Type, "tokenText", fmt.Sprintf("%q", token.Text))
	var n Node
	switch token.Type {
	case itemText:
		n = t.paragraph(token)
	case itemInlineEmphasisOpen:
		t.inlineEmphasis(token)
	case itemInlineStrongOpen:
		t.inlineStrong(token)
	case itemInlineLiteralOpen:
		t.inlineLiteral(token)
	case itemInlineInterpretedTextOpen:
		t.inlineInterpretedText(token)
	case itemInlineInterpretedTextRoleOpen:
		t.inlineInterpretedTextRole(token)
	case itemCommentMark:
		t.comment(token)
	case itemEnumListArabic:
		t.enumList(token)
	case itemSpace:
	case itemBlankLine, itemEscape:
	case itemBlockQuote:
		t.blockquote(token)
	// case itemDefinitionTerm:
	// t.definitionTerm(token)
	// case itemBullet:
	// t.bulletListItem(token)
	default:
		t.log(fmt.Errorf("Token type: %q is not yet supported in the parser", token.Type.String()))
	}
	return n
}

// backup shifts the token buffer right one position.
func (t *Tree) backup() {
	t.token[0] = nil
	for x := len(t.token) - 1; x > 0; x-- {
		t.token[x] = t.token[x-1]
		t.token[x-1] = nil
	}
	if t.token[zed] == nil {
		t.log("Current token is: <nil>")
	} else {
		t.log("Current token is: %T", t.token[zed].Type)
	}
}

// peekBack uses the token buffer to "look back" a number of positions (pos). Looking back more positions than the
// Tree.token buffer allows (3) will generate a panic.
func (t *Tree) peekBack(pos int) *item {
	return t.token[zed-pos]
}

func (t *Tree) peekBackTo(item itemElement) (tok *item) {
	for i := zed - 1; i >= 0; i-- {
		if t.token[i] != nil && t.token[i].Type == item {
			tok = t.token[i]
			break
		}
	}
	return
}

// peek looks ahead in the token stream a number of positions (pos) and gets the next token from the lexer. A pointer to the
// token is kept in the Tree.token buffer. If a token pointer already exists in the buffer, that token is used instead and no
// tokens are received the the lexer stream (channel).
func (t *Tree) peek(pos int) *item {
	nItem := t.token[zed]
	for i := 1; i <= pos; i++ {
		if t.token[zed+i] != nil {
			nItem = t.token[zed+i]
			t.log("msg", "Have token", "token", nItem)
			continue
		} else {
			if t.lex == nil {
				continue
			}
			t.log("Getting next item")
			t.token[zed+i] = t.lex.nextItem()
			nItem = t.token[zed+i]
		}
	}
	return nItem
}

// peekSkip looks ahead one position skipiing a specified itemElement. If that element is found, a pointer is returned,
// otherwise nil is returned.
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

// next is the workhorse of the parser. It is repsonsible for getting the next token from the lexer stream (channel). If the
// next token already exists in the token buffer, than the token buffer is shifted left and the pointer to the "zed" token is
// returned. pos specifies the number of times to call next.
func (t *Tree) next(pos int) *item {
	if pos == 0 {
		return t.token[zed]
	}
	for x := 0; x < len(t.token)-1; x++ {
		t.token[x] = t.token[x+1]
		t.token[x+1] = nil
	}
	if t.token[zed] == nil && t.lex != nil {
		t.token[zed] = t.lex.nextItem()
	}
	pos--
	if pos > 0 {
		t.next(pos)
	}
	return t.token[zed]
}

// clearTokens sets tokens from begin to end to nil.
func (t *Tree) clearTokens(begin, end int) {
	for i := begin; i <= end; i++ {
		t.token[i] = nil
	}
}

// section is responsible for parsing the title, overline, and underline tokens returned from the parser. If there are errors
// parsing these elements, than a systemMessage is generated and added to Tree.Nodes.
func (t *Tree) section(i *item) Node {
	var overAdorn, indent, title, underAdorn *item

	pBack := t.peekBack(1)
	pFor := t.peekSkip(itemSpace)
	tZedLen := t.token[zed].Length

	if pFor != nil && pFor.Type == itemTitle {
		// Section with overline
		pBack := t.peekBack(1)
		// Check for errors
		if tZedLen < 3 && tZedLen != pFor.Length {
			t.next(2)
			bTok := t.peekBack(1)
			if bTok != nil && bTok.Type == itemSpace {
				t.next(2)
				m := infoUnexpectedTitleOverlineOrTransition
				return t.systemMessage(m)
			}
			return t.systemMessage(infoOverlineTooShortForTitle)
		} else if pBack != nil && pBack.Type == itemSpace {
			// Indented section (error) The section title has an indented overline
			m := severeUnexpectedSectionTitleOrTransition
			return t.systemMessage(m)
		}

		overAdorn = i
		t.next(1)

	loop:
		for {
			switch tTok := t.token[zed]; tTok.Type {
			case itemTitle:
				title = tTok
				t.next(1)
			case itemSpace:
				indent = tTok
				t.next(1)
			case itemSectionAdornment:
				underAdorn = tTok
				break loop
			}
		}
	} else if pBack != nil && (pBack.Type == itemTitle || pBack.Type == itemSpace) {
		// Section with no overline Check for errors
		if pBack.Type == itemSpace {
			pBack := t.peekBack(2)
			if pBack != nil && pBack.Type == itemTitle {
				// The section underline is indented
				m := severeUnexpectedSectionTitle
				return t.systemMessage(m)
			}
		} else if tZedLen < 3 && tZedLen != pBack.Length {
			// Short underline
			return t.systemMessage(infoUnderlineTooShortForTitle)
		}
		// Section OKAY
		title = t.peekBack(1)
		underAdorn = i

	} else if pFor != nil && pFor.Type == itemText {
		// If a section contains an itemText, it is because the underline is missing, therefore we generate an
		// error based on what follows the itemText.
		t.next(2) // Move the token buffer past the error tokens
		if tZedLen < 3 && tZedLen != pFor.Length {
			t.backup()
			return t.systemMessage(infoOverlineTooShortForTitle)
		} else if p := t.peek(1); p != nil && p.Type == itemBlankLine {
			m := severeMissingMatchingUnderlineForOverline
			return t.systemMessage(m)
		}
		return t.systemMessage(severeIncompleteSectionTitle)
	} else if pFor != nil && pFor.Type == itemSectionAdornment {
		// Missing section title
		t.next(1) // Move the token buffer past the error token
		return t.systemMessage(errorInvalidSectionOrTransitionMarker)
	} else if pFor != nil && pFor.Type == itemEOF {
		// Missing underline and at EOF
		return t.systemMessage(errorInvalidSectionOrTransitionMarker)
	}

	if overAdorn != nil && overAdorn.Text != underAdorn.Text {
		return t.systemMessage(severeOverlineUnderlineMismatch)
	}

	// Determine the level of the section and where to append it to in t.Nodes
	sec := newSection(title, overAdorn, underAdorn, indent)
	t.log("msg", "Adding section level", "sectionLevel", sec.UnderLine.Rune)

	msg := t.sectionLevels.Add(sec)
	if msg != parserMessageNil {
		t.log("Found inconsistent section level!")
		return t.systemMessage(severeTitleLevelInconsistent)
	}

	sec.Level = t.sectionLevels.lastSectionNode.Level
	if sec.Level == 1 {
		t.log("Setting nodeTarget to Tree.Nodes!")
		t.nodeTarget = &t.Nodes
	} else {
		lSec := t.sectionLevels.lastSectionNode
		if sec.Level > 1 {
			lSec = t.sectionLevels.LastSectionByLevel(sec.Level - 1)
		}
		t.nodeTarget = &lSec.NodeList
	}

	// The following checks have to be made after the SectionNode has been initialized so that any parserMessages can be
	// appended to the SectionNode.NodeList.
	oLen := title.Length
	if indent != nil {
		oLen = indent.Length + title.Length
	}

	if overAdorn != nil && oLen > overAdorn.Length {
		m := warningShortOverline
		sec.NodeList = append(sec.NodeList, t.systemMessage(m))
	} else if overAdorn == nil && title.Length != underAdorn.Length {
		m := warningShortUnderline
		sec.NodeList = append(sec.NodeList, t.systemMessage(m))
	}
	return sec
}

func (t *Tree) comment(i *item) Node {
	var n Node
	if t.peek(1).Type == itemBlankLine {
		t.log("Found empty comment block")
		n := newComment(&item{StartPosition: i.StartPosition, Line: i.Line})
		t.nodeTarget.append(n)
		return n
	}
	if nSpace := t.peek(1); nSpace != nil && nSpace.Type != itemSpace {
		// The comment element itself is valid, but we need to add it to the NodeList before the systemMessage.
		t.log("Missing space after comment mark! (warningExplicitMarkupWithUnIndent)")
		n = newComment(&item{Line: i.Line})
		sm := t.systemMessage(warningExplicitMarkupWithUnIndent)
		t.nodeTarget.append(n, sm)
		return n
	}
	nPara := t.peek(2)
	if nPara != nil && nPara.Type == itemText {
		// Skip the itemSpace
		t.next(2)
		// See if next line is indented, if so, it is part of the comment
		if t.peek(1).Type == itemSpace && t.peek(2).Type == itemText {
			t.log("Found NodeComment block")
			t.next(2)
			for {
				nPara.Text += "\n" + t.token[zed].Text
				if t.peek(1).Type == itemSpace && t.peek(2).Type == itemText {
					t.next(2)
				} else {
					break
				}
			}
			nPara.Length = len(nPara.Text)
		} else if z := t.peek(1); z != nil && z.Type != itemBlankLine && z.Type != itemCommentMark && z.Type != itemEOF {
			// A valid comment contains a blank line after the comment block
			t.log("Found warningExplicitMarkupWithUnIndent")
			n = newComment(nPara)
			sm := t.systemMessage(warningExplicitMarkupWithUnIndent)
			t.nodeTarget.append(n)
			t.nodeTarget.append(sm)
			return n
		} else {
			// Just a regular single lined comment
			t.log("Found one-line NodeComment")
		}
		n = newComment(nPara)
	}
	t.nodeTarget.append(n)
	return n
}

// systemMessage generates a Node based on the passed parserMessage. The generated message is returned as a
// SystemMessageNode.
func (t *Tree) systemMessage(err parserMessage) Node {
	var lbText string
	var lbTextLen int
	var backToken int

	s := newSystemMessage(&item{Type: itemSystemMessage, Line: t.token[zed].Line}, err)
	msg := newText(&item{
		Text:   err.Message(),
		Length: len(err.Message()),
	})

	t.log("msg", "Adding msg to system message NodeList", "systemMessage", err)
	s.NodeList.append(msg)

	var overLine, indent, title, underLine, newLine string

	switch err {
	case infoOverlineTooShortForTitle:
		var inText string
		if t.token[zed-2] != nil {
			inText = t.token[zed-2].Text + "\n" + t.token[zed-1].Text + "\n" + t.token[zed].Text
			s.Line = t.token[zed-2].Line
			t.token[zed-2] = nil
		} else {
			inText = t.token[zed-1].Text + "\n" + t.token[zed].Text
			s.Line = t.token[zed-1].Line
		}
		infoTextLen := len(inText)
		// Modify the token buffer to change the current token to a itemText then backup the token buffer so the
		// next loop gets the new paragraph
		t.token[zed-1] = nil
		t.token[zed].Type = itemText
		t.token[zed].Text = inText
		t.token[zed].Length = infoTextLen
		t.token[zed].Line = s.Line
		t.backup()
	case infoUnexpectedTitleOverlineOrTransition:
		oLin := t.peekBackTo(itemSectionAdornment)
		titl := t.peekBackTo(itemTitle)
		uLin := t.token[zed]
		inText := oLin.Text + "\n" + titl.Text + "\n" + uLin.Text
		s.Line = oLin.Line
		t.clearTokens(zed-4, zed-1)
		infoTextLen := len(inText)
		// Modify the token buffer to change the current token to a itemText then backup the token buffer so the
		// next loop gets the new paragraph
		t.token[zed].Type = itemText
		t.token[zed].Text = inText
		t.token[zed].Length = infoTextLen
		t.token[zed].Line = s.Line
		t.token[zed].StartPosition = oLin.StartPosition
		t.backup()
	case infoUnderlineTooShortForTitle:
		inText := t.token[zed-1].Text + "\n" + t.token[zed].Text
		infoTextLen := len(inText)
		s.Line = t.token[zed-1].Line
		// Modify the token buffer to change the current token to a itemText then backup the token buffer so the
		// next loop gets the new paragraph
		t.token[zed-1] = nil
		t.token[zed].Type = itemText
		t.token[zed].Text = inText
		t.token[zed].Length = infoTextLen
		t.token[zed].Line = s.Line
		t.backup()
	case warningShortOverline, severeOverlineUnderlineMismatch:
		backToken = zed - 2
		if t.peekBack(2).Type == itemSpace {
			backToken = zed - 3
			indent = t.token[zed-2].Text
		}
		overLine = t.token[backToken].Text
		title = t.token[zed-1].Text
		underLine = t.token[zed].Text
		newLine = "\n"
		lbText = overLine + newLine + indent + title + newLine + underLine
		s.Line = t.token[backToken].Line
		lbTextLen = len(lbText)
	case warningShortUnderline, severeUnexpectedSectionTitle:
		backToken = zed - 1
		if t.peekBack(1).Type == itemSpace {
			backToken = zed - 2
		}
		lbText = t.token[backToken].Text + "\n" + t.token[zed].Text
		lbTextLen = len(lbText)
		s.Line = t.token[zed-1].Line
	case warningExplicitMarkupWithUnIndent:
		s.Line = t.token[zed+1].Line
	case errorInvalidSectionOrTransitionMarker:
		lbText = t.token[zed-1].Text + "\n" + t.token[zed].Text
		s.Line = t.token[zed-1].Line
		lbTextLen = len(lbText)
	case severeIncompleteSectionTitle,
		severeMissingMatchingUnderlineForOverline:
		lbText = t.token[zed-2].Text + "\n" + t.token[zed-1].Text + t.token[zed].Text
		s.Line = t.token[zed-2].Line
		lbTextLen = len(lbText)
	case severeUnexpectedSectionTitleOrTransition:
		lbText = t.token[zed].Text
		lbTextLen = len(lbText)
		s.Line = t.token[zed].Line
	case severeTitleLevelInconsistent:
		if t.peekBack(2).Type == itemSectionAdornment {
			lbText = t.token[zed-2].Text + "\n" + t.token[zed-1].Text + "\n" + t.token[zed].Text
			lbTextLen = len(lbText)
			s.Line = t.token[zed-2].Line
		} else {
			lbText = t.token[zed-1].Text + "\n" + t.token[zed].Text
			lbTextLen = len(lbText)
			s.Line = t.token[zed-1].Line
		}
	}

	if lbTextLen > 0 {
		lb := newLiteralBlock(&item{Type: itemLiteralBlock, Text: lbText, Length: lbTextLen})
		s.NodeList = append(s.NodeList, lb)
	}

	t.Messages.append(s)

	return s
}

var lastEnum *EnumListNode

func (t *Tree) enumList(i *item) (n Node) {
	// FIXME: This function is COMPLETELY not final. It is only setup for passing section test TitleNumberedGood0100.
	var eNode *EnumListNode
	// var affix *item
	// FIXME: This has been commented out because newParagraph has been deprecated.
	// if lastEnum == nil {
	// t.next(1)
	// affix = t.token[zed]
	// t.next(1)
	// eNode = newEnumListNode(i, affix)
	// t.next(1)
	// eNode.NodeList.append(newParagraph(t.token[zed]))
	// } else {
	// t.next(3)
	// lastEnum.NodeList.append(newParagraph(t.token[zed]))
	// return nil
	// }
	lastEnum = eNode
	return eNode
}

func (t *Tree) paragraph(i *item) Node {
	t.log("msg", "Have token", "token", i)
	np := newParagraph()
	t.nodeTarget.append(np)
	t.nodeTarget = &np.NodeList
	nt := newText(i)
	t.nodeTarget.append(nt)
outer:
	// Paragraphs can contain many different types of elements, so we'll need to loop until blank line or nil
	for {
		ci := t.next(1)     // current item
		pi := t.peekBack(1) // previous item
		// ni := t.peek(1)     // next item

		t.log("msg", "Have token", "token", ci)
		if ci == nil {
			t.log("ci == nil, breaking")
			break
		} else if ci.Type == itemEOF {
			t.log("msg", "current item type == itemEOF")
			break
		} else if pi != nil && pi.Type == itemText && ci.Type == itemText {
			t.log("msg", "Previous type == itemText, current type == itemText; Concatenating text!")
			nt.Text += "\n" + ci.Text
			nt.Length = len(nt.Text)
			continue
		}

		t.log("msg", "Going into subparser...")

		switch ci.Type {
		case itemText:
			if pi != nil && pi.Type == itemEscape && pi.StartPosition.Int() > ci.StartPosition.Int() {
				// Parse Test 02.00.01.00 :: Catch escapes at the end of lines
				nt.Text += ci.Text
				nt.Length = len(nt.Text)
			} else {
				nt = newText(ci)
				t.nodeTarget.append(nt)
			}
		case itemInlineEmphasisOpen:
			t.inlineEmphasis(ci)
		case itemInlineStrongOpen:
			t.inlineStrong(ci)
		case itemInlineLiteralOpen:
			t.inlineLiteral(ci)
		case itemInlineInterpretedTextOpen:
			t.inlineInterpretedText(ci)
		case itemInlineInterpretedTextRoleOpen:
			t.inlineInterpretedTextRole(ci)
		case itemCommentMark:
			t.comment(ci)
		// case itemEnumListArabic:
		// t.enumList(ci)
		case itemBlankLine:
			t.log("Found newline, closing paragraph")
			t.backup()
			break outer
		}
		t.log("msg", "Continuing...")
	}
	t.log("t.indents.len", t.indents.len())
	if t.indents.len() > 0 {
		t.nodeTarget = t.indents.topNodeList()
	} else {
		t.nodeTarget = &t.Nodes
	}
	return np
}

func (t *Tree) inlineEmphasis(i *item) {
	t.next(1)
	t.nodeTarget.append(newInlineEmphasis(t.token[zed]))
	t.next(1)
}

func (t *Tree) inlineStrong(i *item) {
	t.next(1)
	t.nodeTarget.append(newInlineStrong(t.token[zed]))
	t.next(1)
}

func (t *Tree) inlineLiteral(i *item) {
	t.next(1)
	t.nodeTarget.append(newInlineLiteral(t.token[zed]))
	t.next(1)
}

func (t *Tree) inlineInterpretedText(i *item) {
	t.next(1)
	n := newInlineInterpretedText(t.token[zed])
	t.nodeTarget.append(n)
	t.next(1)
	if t.peek(1).Type == itemInlineInterpretedTextRoleOpen {
		t.next(2)
		n.NodeList.append(newInlineInterpretedTextRole(t.token[zed]))
		t.next(1)
	}
}

func (t *Tree) inlineInterpretedTextRole(i *item) {
	t.next(1)
	t.nodeTarget.append(newInlineInterpretedTextRole(t.token[zed]))
	t.next(1)
}

func (t *Tree) blockquote(i *item) {
	//
	//  FIXME: Blockquote parsing is NOT fully implemented.
	//
	sec := newBlockQuote(i)
	t.Nodes.append(sec)
}

func (t *Tree) definitionTerm(i *item) Node {
	//
	//  FIXME: Definition list parsing is NOT fully implemented.
	//
	dl := newDefinitionList(&item{Line: i.Line})
	t.nodeTarget.append(dl)
	t.nodeTarget = &dl.NodeList
	t.next(1)

	// Container for definition items
	dli := newDefinitionListItem(i, t.peek(1))
	t.nodeTarget.append(dli)
	t.nodeTarget = &dli.Definition.NodeList

	// Gather definitions and body elements
	for {
		ni := t.next(1)
		if ni == nil {
			break
		}
		t.log("msg", "Have token", "token", ni)
		pb := t.peekBack(1)
		if ni.Type == itemSpace {
			t.log("msg", "continue; ni.Type == itemSpace")
			continue
		} else if ni.Type == itemEOF {
			t.log("msg", "break; ni.Type == itemEOF")
			break
		} else if ni.Type == itemBlankLine {
			t.log("Setting nodeTarget to dli")
			t.nodeTarget = &dli.Definition.NodeList
		} else if ni.Type == itemCommentMark && (pb != nil && pb.Type != itemSpace) {
			// Comment at start of the line breaks current definition list
			t.log("Have itemCommentMark at start of the line!")
			t.nodeTarget = &t.Nodes
			t.backup()
			break
		} else if ni.Type == itemDefinitionText {
			// FIXME: This function is COMPLETELY not final. It is only setup for passing section test TitleNumberedGood0100.
			np := newParagraphWithNodeText(ni)
			t.nodeTarget.append(np)
			t.nodeTarget = &np.NodeList
			t.log("msg", "continue; ni.Type == itemDefinitionText")
			continue
		} else if ni.Type == itemDefinitionTerm {
			dli2 := newDefinitionListItem(ni, t.peek(2))
			t.nodeTarget = &dl.NodeList
			t.nodeTarget.append(dli2)
			t.nodeTarget = &dli2.Definition.NodeList
			t.log("msg", "continue; ni.Type == itemDefinitionTerm")
			continue
		}
		t.subParseBodyElements(ni)
	}
	return dl
}

func (t *Tree) bulletList(i *item) {
	//
	// FIXME: Bullet List Parsing is NOT fully implemented
	//
	bl := newBulletListNode(i)
	t.openList = bl
	t.nodeTarget.append(bl)
	t.nodeTarget = &bl.NodeList

	// Get the bullet list paragraph
	t.next(1)
	bli := newBulletListItemNode(i)

	// Set the node target to the bullet list paragraph
	t.nodeTarget.append(bli)
	t.nodeTarget = &bli.NodeList
	t.indents.add(i, bli)

	// Capture all bullet items until un-indent
	for {
		ni := t.next(1)
		t.log("msg", "Have token", "token", fmt.Sprintf("%+#v", ni))
		if ni == nil {
			t.log("break next item == nil")
			break
		} else if ni.Type == itemEOF {
			t.log("break itemEOF")
			break
		} else if t.indents.len() > 0 && len(*t.indents.topNodeList()) > 0 && t.peekBack(1).Type == itemSpace &&
			t.peekBack(2).Type != itemCommentMark {
			t.log("msg", "Have indents",
				"lastStartPosition", t.indents.lastStartPosition(),
				"ni.StartPosition", ni.StartPosition)
			if t.indents.lastStartPosition() != ni.StartPosition {
				// FIXME: WE SHOULD NEVER EXIT IN LIBRARY !! This is just debug code, but we need to add
				// proper handling for this ...
				t.log(errors.New("Unexpected un-indent!"))
				spd.Dump(t.indents)
				os.Exit(1)
			}
		}

		t.subParseBodyElements(ni)
	}
	t.indents.pop()
}
