package parser

import (
	"fmt"

	"golang.org/x/text/unicode/norm"

	"github.com/demizer/go-rst/pkg/log"
	"github.com/demizer/go-rst/pkg/testutil"

	klog "github.com/go-kit/kit/log"

	doc "github.com/demizer/go-rst/pkg/document"
	tok "github.com/demizer/go-rst/pkg/token"
)

const (
	// The middle of the Parser.token buffer so that there are three possible "backup" token positions and three
	// forward "peek" positions.
	zed = 4

	// Default indent width
	indentWidth = 4
)

// Parser contains the parser Parser. The Nodes field contains the parsed nodes of the input input data.
type Parser struct {
	Name     string        // The name of the current parser input
	Nodes    *doc.NodeList // The root node list
	Messages doc.NodeList  // Messages generated by the parser

	nodeTarget *doc.NodeTarget // Used to append nodes to a target NodeList
	text       string          // The input text
	lex        *tok.Lexer      // The place where tokens come from
	token      [9]*tok.Item    // Token buffer, number 4 is the middle. AKA the "zed" token
	indents    *indentQueue    // Indent level tracking

	bqLevel *doc.BlockQuoteNode // FIXME: will be replaced with blockquoteLevels

	sectionLevels *sectionLevels     // Encountered section levels
	sections      []*doc.SectionNode // Pointers to encountered sections

	openList doc.Node // Open Bullet List, Enum List, or Definition List

	log.Logger
}

// New returns a fresh parser Parser.
func New(name, text string, logr klog.Logger) *Parser {
	nl := make(doc.NodeList, 0)
	t := &Parser{
		Name:          name,
		Nodes:         &nl,
		text:          text,
		sectionLevels: newSectionLevels(testutil.StdLogger),
		indents:       new(indentQueue),
		nodeTarget:    doc.NewNodeTarget(&nl, logr),
		Logger:        log.NewLogger("parser", true, testutil.LogExcludes, logr),
	}
	t.Msgr("Parser.Nodes pointer", "nodeListPointer", fmt.Sprintf("%p", nl))
	return t
}

// Parse is the entry point for the reStructuredText parser. Errors generated by the parser are returned as a NodeList.
func Parse(name, text string, logr klog.Logger) (p *Parser, errors doc.NodeList) {
	p = New(name, text, logr)
	if !norm.NFC.IsNormalString(text) {
		text = norm.NFC.String(text)
	}
	p.Parse(text)
	errors = p.Messages
	return
}

// startParse initializes the parser, using the lexer.
func (p *Parser) startParse(lex *tok.Lexer) {
	p.lex = lex
}

// Parse activates the parser using text as input data. A parse Parser is returned on success or failure. Users of the
// Parse package should use the Top level Parse function.
func (p *Parser) Parse(text string) (*Parser, error) {
	l, err := tok.Lex(p.Name, []byte(text), p.StdLogger())
	if err != nil {
		err = fmt.Errorf("parsing error: %s", err)
	}
	p.startParse(l)
	p.text = text
	p.parse()
	return p, err
}

// parse is where items are retrieved from the parser and dispatched according to the itemElement type.
func (p *Parser) parse() {
	for {
		var n doc.Node

		token := p.next(1)
		if token == nil || token.Type == tok.EOF {
			break
		}

		p.Msgr("Parser got token", "token", token)

		switch token.Type {
		case tok.Text:
			p.paragraph(token)
		case tok.InlineEmphasisOpen:
			p.inlineEmphasis(token)
		case tok.InlineStrongOpen:
			p.inlineStrong(token)
		case tok.InlineLiteralOpen:
			p.inlineLiteral(token)
		case tok.InlineInterpretedTextOpen:
			p.inlineInterpretedText(token)
		case tok.InlineInterpretedTextRoleOpen:
			p.inlineInterpretedTextRole(token)
		case tok.Transition:
			// FIXME: Workaround until transitions are supported
			p.nodeTarget.Append(doc.NewTransition(token))
		case tok.CommentMark:
			p.comment(token)
		case tok.SectionAdornment:
			p.section(token)
		case tok.EnumListArabic:
			n = p.enumList(token)
			// FIXME: This is only until enumerated list are properly implemented.
			if n == nil {
				continue
			}
			p.nodeTarget.Append(n)
		case tok.Space:
			//
			//  FIXME: Blockquote parsing is NOT fully implemented.
			//
			if p.peekBack(1).Type == tok.BlankLine && p.bqLevel == nil {
				// Ignore if next item is a blockquote from the lexer
				if pn := p.peek(1); pn != nil && pn.Type == tok.BlockQuote {
					p.Msg("Next item is blockquote; not creating empty blockquote")
					continue
				}
				p.Msg("Creating empty blockquote!")
				p.emptyblockquote(token)
			} else if p.peekBack(1).Type == tok.BlankLine {
				p.nodeTarget.SetParent(p.bqLevel)
			}
		case tok.BlankLine, tok.Title, tok.Escape:
			// itemTitle is consumed when evaluating SectionAdornment
			continue
		case tok.BlockQuote:
			p.blockquote(token)
		case tok.DefinitionTerm:
			p.definitionTerm(token)
		case tok.Bullet:
			p.bulletList(token)
		default:
			p.Msg(fmt.Sprintf("Token type: %q is not yet supported in the parser", token.Type.String()))
		}

	}
}

func (p *Parser) subParseBodyElements(token *tok.Item) doc.Node {
	p.Msgr("Have token", "tokenType", token.Type, "tokenText", fmt.Sprintf("%q", token.Text))
	var n doc.Node
	switch token.Type {
	case tok.Text:
		n = p.paragraph(token)
	case tok.InlineEmphasisOpen:
		p.inlineEmphasis(token)
	case tok.InlineStrongOpen:
		p.inlineStrong(token)
	case tok.InlineLiteralOpen:
		p.inlineLiteral(token)
	case tok.InlineInterpretedTextOpen:
		p.inlineInterpretedText(token)
	case tok.InlineInterpretedTextRoleOpen:
		p.inlineInterpretedTextRole(token)
	case tok.CommentMark:
		p.comment(token)
	case tok.EnumListArabic:
		p.enumList(token)
	case tok.Space:
	case tok.BlankLine, tok.Escape:
	case tok.BlockQuote:
		p.blockquote(token)
	default:
		p.Msg(fmt.Sprintf("Token type: %q is not yet supported in the parser", token.Type.String()))
	}
	return n
}

// backup shifts the token buffer right one position.
func (p *Parser) backup() {
	p.token[0] = nil
	for x := len(p.token) - 1; x > 0; x-- {
		p.token[x] = p.token[x-1]
		p.token[x-1] = nil
	}
	if p.token[zed] == nil {
		p.Msg("Current token is: <nil>")
	} else {
		p.Msg(fmt.Sprintf("Current token is: %T", p.token[zed].Type))
	}
}

// peekBack uses the token buffer to "look back" a number of positions (pos). Looking back more positions than the
// Parser.token
// buffer allows (3) will generate a panic.
func (p *Parser) peekBack(pos int) *tok.Item {
	return p.token[zed-pos]
}

func (p *Parser) peekBackTo(item tok.Type) (tok *tok.Item) {
	for i := zed - 1; i >= 0; i-- {
		if p.token[i] != nil && p.token[i].Type == item {
			tok = p.token[i]
			break
		}
	}
	return
}

// peek looks ahead in the token stream a number of positions (pos) and gets the next token from the lexer. A pointer to the
// token is kept in the Parser.token buffer. If a token pointer already exists in the buffer, that token is used instead
// and no tokens are received the the lexer stream (channel).
func (p *Parser) peek(pos int) *tok.Item {
	nItem := p.token[zed]
	for i := 1; i <= pos; i++ {
		if p.token[zed+i] != nil {
			nItem = p.token[zed+i]
			p.Msgr("Have token", "token", nItem)
			continue
		} else {
			if p.lex == nil {
				continue
			}
			p.Msg("Getting next item")
			p.token[zed+i] = p.lex.NextItem()
			nItem = p.token[zed+i]
		}
	}
	return nItem
}

// peekSkip looks ahead one position skipiing a specified itemElement. If that element is found, a pointer is returned,
// otherwise nil is returned.
func (p *Parser) peekSkip(iSkip tok.Type) *tok.Item {
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

// next token already exists in the token buffer, than the token buffer is shifted left and the pointer to the "zed" token is
// returned. pos specifies the number of times to call next.
func (p *Parser) next(pos int) *tok.Item {
	if pos == 0 {
		return p.token[zed]
	}
	for x := 0; x < len(p.token)-1; x++ {
		p.token[x] = p.token[x+1]
		p.token[x+1] = nil
	}
	if p.token[zed] == nil && p.lex != nil {
		p.token[zed] = p.lex.NextItem()
	}
	pos--
	if pos > 0 {
		p.next(pos)
	}
	return p.token[zed]
}

// clearTokens sets tokens from begin to end to nil.
func (p *Parser) clearTokens(begin, end int) {
	for i := begin; i <= end; i++ {
		p.token[i] = nil
	}
}
