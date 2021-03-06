package document

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/demizer/go-rst/pkg/messages"
	tok "github.com/demizer/go-rst/pkg/token"
)

// NodeType identifies the type of a parse tree node.
type NodeType int

const (
	// NodeSection is a section element.
	NodeSection NodeType = iota

	// NodeText is ordinary text.
	NodeText

	// NodeParagraph is a paragraph container that contains text and inline markup.
	NodeParagraph

	// NodeAdornment is the overline or underline of a section.
	NodeAdornment

	// NodeBlockQuote is a blockquote element.
	NodeBlockQuote

	// NodeSystemMessage contains an error encountered by the parser.
	NodeSystemMessage

	// NodeSystemMessages contains a list of NodeSystemMessage
	NodeSystemMessages

	// NodeLiteralBlock is a literal block element.
	NodeLiteralBlock

	// NodeTransition is a transition element. Transitions are very similar
	// to NodeSection except that they have newlines before and after.
	NodeTransition

	// NodeTitle is a section title element to be used inside SectionNodes.
	NodeTitle

	// NodeComment is a comment element
	NodeComment

	// NodeBulletList is the beginning of a bullet list
	NodeBulletList

	// NodeBulletListItem is a bullet list item
	NodeBulletListItem

	// NodeEnumList is an enumerated list
	NodeEnumList

	// NodeDefinitionList is the beginning of a definition list element
	NodeDefinitionList

	// NodeDefinitionListItem is a definition list item
	NodeDefinitionListItem

	// NodeDefinitionTerm is a definition list term element
	NodeDefinitionTerm

	// NodeDefinition is a definition element
	NodeDefinition

	// NodeInlineEmphasis is the italicized text element
	NodeInlineEmphasis

	// NodeInlineStrong is the bold text element
	NodeInlineStrong

	// NodeInlineLiteral defines inline literal markup
	NodeInlineLiteral

	// NodeInlineInterpretedText is part of an interpreted text role
	NodeInlineInterpretedText

	// NodeInlineInterpretedTextRole is the role of the interpreted text
	NodeInlineInterpretedTextRole
)

var nodeTypes = [...]string{
	"NodeSection",
	"NodeText",
	"NodeParagraph",
	"NodeAdornment",
	"NodeBlockQuote",
	"NodeSystemMessage",
	"NodeSystemMessages",
	"NodeLiteralBlock",
	"NodeTransition",
	"NodeTitle",
	"NodeComment",
	"NodeBulletList",
	"NodeBulletListItem",
	"NodeEnumList",
	"NodeDefinitionList",
	"NodeDefinitionListItem",
	"NodeDefinitionTerm",
	"NodeDefinition",
	"NodeInlineEmphasis",
	"NodeInlineStrong",
	"NodeInlineLiteral",
	"NodeInlineInterpretedText",
	"NodeInlineInterpretedTextRole",
}

// Type returns the type of a node element.
func (n NodeType) Type() NodeType { return n }

func (n NodeType) String() string { return nodeTypes[n] }

// Node is the interface used to implement parser nodes.
type Node interface {
	NodeType() NodeType
	String() string
}

// EnumListType identifies the type of the enumeration list element
type EnumListType int

const (
	enumListArabic EnumListType = iota
	enumListUpperAlpha
	enumListLowerAlpha
	enumListUpperRoman
	enumListLowerRoman
	enumListAuto
)

var enumListTypes = [...]string{
	"enumListArabic",
	"enumListUpperAlpha",
	"enumListLowerAlpha",
	"enumListUpperRoman",
	"enumListLowerRoman",
	"enumListAuto",
}

func (e EnumListType) String() string { return enumListTypes[e] }

// EnumAffixType identifies the type of affix for the Enumerated list element
type EnumAffixType int

const (
	enumAffixPeriod EnumAffixType = iota
	enumAffixParenthesisSurround
	enumAffixParenthesisRight
)

var enumAffixesTypes = [...]string{
	"enumAffixPeriod",
	"enumAffixParenthesisSurround",
	"enumAffixParenthesisRight",
}

// String satisfies the Stringer interface
func (a EnumAffixType) String() string { return enumAffixesTypes[a] }

// SectionNode is a a single section node. It contains overline, title, and underline nodes. NodeList contains nodes that are
// children of the section.
type SectionNode struct {
	Type NodeType `json:"type,string"`

	// Level is the hierarchical level of the section. The first level is level 1, any further sections encountered after
	// the first level are given consecutive level numbers.
	Level int `json:"level"`

	// OverLine and UnderLine are the parsed Nodes that make up the section.
	Title     *TitleNode     `json:"title"`
	OverLine  *AdornmentNode `json:"overLine"`
	UnderLine *AdornmentNode `json:"underLine"`

	// NodeList contains
	NodeList `json:"nodeList"`
}

// NodeType returns the Node type of the SectionNode.
func (s *SectionNode) NodeType() NodeType { return s.Type }

// String satisfies the Stringer interface
func (s *SectionNode) String() string { return fmt.Sprintf("%#v", s) }

func NewSection(title *TitleNode, overSec *tok.Item, underSec *tok.Item, indent *tok.Item) *SectionNode {
	var indentLen int
	n := &SectionNode{Type: NodeSection}
	if indent != nil {
		indentLen = indent.Length
	}
	if title != nil {
		n.Title = title
		n.Title.IndentLength = indentLen
	}

	if overSec != nil && overSec.Text != "" {
		Rune := rune(overSec.Text[0])
		n.OverLine = &AdornmentNode{
			Type:          NodeAdornment,
			Rune:          Rune,
			StartPosition: overSec.StartPosition,
			Line:          overSec.Line,
			Length:        overSec.Length,
		}
	}
	Rune := rune(underSec.Text[0])
	n.UnderLine = &AdornmentNode{
		Rune:          Rune,
		Type:          NodeAdornment,
		StartPosition: underSec.StartPosition,
		Line:          underSec.Line,
		Length:        underSec.Length,
	}

	return n
}

// MarshalJSON satisfies the Marshaler interface.
func (s SectionNode) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString("{")
	buffer.WriteString(fmt.Sprintf("\"type\": %q,", s.Type.String()))
	buffer.WriteString(fmt.Sprintf("\"level\": %d,", s.Level))

	t, err := json.Marshal(s.Title)
	if err != nil {
		return nil, err
	}
	buffer.WriteString(fmt.Sprintf("\"title\": %s,", string(t)))

	o, err := json.Marshal(s.OverLine)
	if err != nil {
		return nil, err
	}
	buffer.WriteString(fmt.Sprintf("\"overLine\": %s,", string(o)))

	u, err := json.Marshal(s.UnderLine)
	if err != nil {
		return nil, err
	}
	buffer.WriteString(fmt.Sprintf("\"underLine\": %s,", string(u)))

	b, err := json.Marshal(s.NodeList)
	if err != nil {
		return nil, err
	}
	if string(b) == "null" {
		b = []byte{'[', ' ', ']'}
	}
	buffer.WriteString(fmt.Sprintf("\"nodeList\": %s", string(b)))
	buffer.WriteString("}")

	return buffer.Bytes(), nil
}

// TitleNode contains the parsed data for a section titles.
type TitleNode struct {
	// Text          string   `json:"text"`
	Type          NodeType          `json:"type"`
	IndentLength  int               `json:"indentLength,omitempty"`
	Length        int               `json:"length"`
	Line          int               `json:"line,omitempty"`
	StartPosition int               `json:"startPosition,omitempty"`
	NodeList      `json:"nodeList"` // NodeList contains children of the ParagraphNode, even other ParagraphNodes!
}

// NodeType returns the Node type of the TitleNode.
func (t TitleNode) NodeType() NodeType { return t.Type }

// String satisfies the Stringer interface
func (t TitleNode) String() string { return fmt.Sprintf("%#v", t) }

// MarshalJSON satisfies the Marshaler interface.
func (t TitleNode) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString("{")
	buffer.WriteString(fmt.Sprintf("\"type\": %q,", t.Type.String()))
	if t.IndentLength > 0 {
		buffer.WriteString(fmt.Sprintf("\"indentLength\": %d,", t.IndentLength))
	}
	buffer.WriteString(fmt.Sprintf("\"length\": %d,", t.Length))
	buffer.WriteString(fmt.Sprintf("\"line\": %d,", t.Line))
	buffer.WriteString(fmt.Sprintf("\"startPosition\": %d,", t.StartPosition))

	b, err := json.Marshal(t.NodeList)
	if err != nil {
		return nil, err
	}
	if string(b) == "null" {
		b = []byte{'[', ' ', ']'}
	}
	buffer.WriteString(fmt.Sprintf("\"nodeList\": %s", string(b)))
	buffer.WriteString("}")

	return buffer.Bytes(), nil
}

func NewTitleNode() *TitleNode { return &TitleNode{Type: NodeTitle} }

func NewTitleNodeWithText(i *tok.Item) *TitleNode {
	tn := &TitleNode{
		Type:          NodeTitle,
		Length:        i.Length,
		StartPosition: i.StartPosition,
		Line:          i.Line,
	}
	tn.Append(NewText(i))
	return tn
}

// AdornmentNode contains the parsed data for a section overline or underline.
type AdornmentNode struct {
	Type          NodeType `json:"type"`
	Rune          rune     `json:"rune,string"`
	Length        int      `json:"length"`
	Line          int      `json:"line,omitempty"`
	StartPosition int      `json:"startPosition,omitempty"`
}

// NodeType returns the Node type of the AdornmentNode.
func (a AdornmentNode) NodeType() NodeType { return a.Type }

// String satisfies the Stringer interface
func (a AdornmentNode) String() string { return fmt.Sprintf("%#v", a) }

// MarshalJSON satisfies the Marshaler interface.
func (a AdornmentNode) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type          string `json:"type"`
		Rune          string `json:"rune"`
		Length        int    `json:"length"`
		Line          int    `json:"line,omitempty"`
		StartPosition int    `json:"startPosition,omitempty"`
	}{
		Type:          nodeTypes[a.Type],
		Rune:          string(a.Rune),
		Length:        a.Length,
		Line:          a.Line,
		StartPosition: a.StartPosition,
	})
}

// TextNode is ordinary text. Typically added to the nodelist of parapgraphs.
type TextNode struct {
	Type          NodeType `json:"type"`
	Text          string   `json:"text"`
	Length        int      `json:"length"`
	Line          int      `json:"line,omitempty"`
	StartPosition int      `json:"startPosition,omitempty"`
}

func NewText(i *tok.Item) *TextNode {
	return &TextNode{
		Type:          NodeText,
		Text:          i.Text,
		Length:        i.Length,
		Line:          i.Line,
		StartPosition: i.StartPosition,
	}
}

// NodeType returns the Node type of the TextNode.
func (t TextNode) NodeType() NodeType { return t.Type }

// String satisfies the Stringer interface
func (t TextNode) String() string { return fmt.Sprintf("%#v", t) }

// MarshalJSON satisfies the Marshaler interface.
func (t TextNode) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type          string `json:"type"`
		Text          string `json:"text"`
		Length        int    `json:"length"`
		Line          int    `json:"line,omitempty"`
		StartPosition int    `json:"startPosition,omitempty"`
	}{
		Type:          nodeTypes[t.Type],
		Text:          t.Text,
		Length:        t.Length,
		Line:          t.Line,
		StartPosition: t.StartPosition,
	})
}

// ParagraphNode is a parsed paragraph.
type ParagraphNode struct {
	Type     NodeType          `json:"type"`
	NodeList `json:"nodeList"` // NodeList contains children of the ParagraphNode, even other ParagraphNodes!
}

func NewParagraph() *ParagraphNode { return &ParagraphNode{Type: NodeParagraph} }

func NewParagraphWithNodeText(i *tok.Item) *ParagraphNode {
	pn := &ParagraphNode{Type: NodeParagraph}
	pn.Append(NewText(i))
	return pn
}

// NodeType returns the Node type of the ParagraphNode.
func (p ParagraphNode) NodeType() NodeType { return p.Type }

// String satisfies the Stringer interface
func (p ParagraphNode) String() string { return fmt.Sprintf("%#v", p) }

// MarshalJSON satisfies the Marshaler interface.
func (p ParagraphNode) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString("{")
	buffer.WriteString(fmt.Sprintf("\"type\": %q,", p.Type.String()))
	b, err := json.Marshal(p.NodeList)
	if err != nil {
		return nil, err
	}
	if string(b) == "null" {
		b = []byte{'[', ' ', ']'}
	}
	buffer.WriteString(fmt.Sprintf("\"nodeList\": %s", string(b)))
	buffer.WriteString("}")
	return buffer.Bytes(), nil
}

// InlineEmphasisNode is parsed inline italicized text.
type InlineEmphasisNode struct {
	Type          NodeType `json:"type"`
	Text          string   `json:"text"`
	Length        int      `json:"length"`
	Line          int      `json:"line,omitempty"`
	StartPosition int      `json:"startPosition,omitempty"`
}

func NewInlineEmphasis(i *tok.Item) *InlineEmphasisNode {
	return &InlineEmphasisNode{
		Type:          NodeInlineEmphasis,
		Text:          i.Text,
		Length:        i.Length,
		Line:          i.Line,
		StartPosition: i.StartPosition,
	}
}

// NodeType returns the Node type of the InlineEmphasisNode.
func (e InlineEmphasisNode) NodeType() NodeType { return e.Type }

// String satisfies the Stringer interface
func (e InlineEmphasisNode) String() string { return fmt.Sprintf("%#v", e) }

// MarshalJSON satisfies the Marshaler interface.
func (e InlineEmphasisNode) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type          string `json:"type"`
		Text          string `json:"text"`
		Length        int    `json:"length"`
		Line          int    `json:"line,omitempty"`
		StartPosition int    `json:"startPosition,omitempty"`
	}{
		Type:          nodeTypes[e.Type],
		Text:          e.Text,
		Length:        e.Length,
		Line:          e.Line,
		StartPosition: e.StartPosition,
	})
}

// InlineStrongNode is a parsed inline bold text.
type InlineStrongNode struct {
	Type          NodeType `json:"type"`
	Text          string   `json:"text"`
	Length        int      `json:"length"`
	Line          int      `json:"line,omitempty"`
	StartPosition int      `json:"startPosition,omitempty"`
}

func NewInlineStrong(i *tok.Item) *InlineStrongNode {
	return &InlineStrongNode{
		Type:          NodeInlineStrong,
		Text:          i.Text,
		Length:        i.Length,
		Line:          i.Line,
		StartPosition: i.StartPosition,
	}
}

// NodeType returns the Node type of the InlineStrongNode.
func (s InlineStrongNode) NodeType() NodeType { return s.Type }

// String satisfies the Stringer interface
func (s InlineStrongNode) String() string { return fmt.Sprintf("%#v", s) }

// MarshalJSON satisfies the Marshaler interface.
func (s InlineStrongNode) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type          string `json:"type"`
		Text          string `json:"text"`
		Length        int    `json:"length"`
		Line          int    `json:"line,omitempty"`
		StartPosition int    `json:"startPosition,omitempty"`
	}{
		Type:          nodeTypes[s.Type],
		Text:          s.Text,
		Length:        s.Length,
		Line:          s.Line,
		StartPosition: s.StartPosition,
	})
}

// InlineLiteralNode is a parsed inline literal node.
type InlineLiteralNode struct {
	Type          NodeType `json:"type"`
	Text          string   `json:"text"`
	Length        int      `json:"length"`
	Line          int      `json:"line,omitempty"`
	StartPosition int      `json:"startPosition,omitempty"`
}

func NewInlineLiteral(i *tok.Item) *InlineLiteralNode {
	return &InlineLiteralNode{
		Type:          NodeInlineLiteral,
		Text:          i.Text,
		Length:        i.Length,
		Line:          i.Line,
		StartPosition: i.StartPosition,
	}
}

// NodeType returns the Node type of the InlineStrongNode.
func (l InlineLiteralNode) NodeType() NodeType { return l.Type }

// String satisfies the Stringer interface
func (l InlineLiteralNode) String() string { return fmt.Sprintf("%#v", l) }

// MarshalJSON satisfies the Marshaler interface.
func (l InlineLiteralNode) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type          string `json:"type"`
		Text          string `json:"text"`
		Length        int    `json:"length"`
		Line          int    `json:"line,omitempty"`
		StartPosition int    `json:"startPosition,omitempty"`
	}{
		Type:          nodeTypes[l.Type],
		Text:          l.Text,
		Length:        l.Length,
		Line:          l.Line,
		StartPosition: l.StartPosition,
	})
}

// InlineInterpretedText is a parsed interpreted text role.
type InlineInterpretedText struct {
	Type          NodeType `json:"type"`
	Text          string   `json:"text"`
	Length        int      `json:"length"`
	Line          int      `json:"line,omitempty"`
	StartPosition int      `json:"startPosition,omitempty"`
	// NodeList contains Nodes parsed as children of the BlockQuoteNode.
	NodeList `json:"nodeList"`
}

func NewInlineInterpretedText(i *tok.Item) *InlineInterpretedText {
	return &InlineInterpretedText{
		Type:          NodeInlineInterpretedText,
		Text:          i.Text,
		Length:        i.Length,
		Line:          i.Line,
		StartPosition: i.StartPosition,
	}
}

// NodeType returns the Node type of the InlineInterpretedText.
func (i InlineInterpretedText) NodeType() NodeType { return i.Type }

// String satisfies the Stringer interface
func (i InlineInterpretedText) String() string { return fmt.Sprintf("%#v", i) }

// MarshalJSON satisfies the Marshaler interface.
func (i InlineInterpretedText) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString("{")
	buffer.WriteString(fmt.Sprintf("\"type\": %q,", i.Type.String()))
	buffer.WriteString(fmt.Sprintf("\"text\": %q,", i.Text))
	buffer.WriteString(fmt.Sprintf("\"length\": %d,", i.Length))
	buffer.WriteString(fmt.Sprintf("\"line\": %d,", i.Line))
	buffer.WriteString(fmt.Sprintf("\"startPosition\": %d,", i.StartPosition))
	b, err := json.Marshal(i.NodeList)
	if err != nil {
		return nil, err
	}
	if string(b) == "null" {
		b = []byte{'[', ' ', ']'}
	}
	buffer.WriteString(fmt.Sprintf("\"nodeList\": %s", string(b)))
	buffer.WriteString("}")
	return buffer.Bytes(), nil
}

// InlineInterpretedTextRole is a parsed interpreted text role.
type InlineInterpretedTextRole struct {
	Type          NodeType `json:"type"`
	Text          string   `json:"text"`
	Length        int      `json:"length"`
	Line          int      `json:"line,omitempty"`
	StartPosition int      `json:"startPosition,omitempty"`
}

func NewInlineInterpretedTextRole(i *tok.Item) *InlineInterpretedTextRole {
	return &InlineInterpretedTextRole{
		Type:          NodeInlineInterpretedTextRole,
		Text:          i.Text,
		Length:        i.Length,
		Line:          i.Line,
		StartPosition: i.StartPosition,
	}
}

// NodeType returns the Node type of the InlineInterpretedTextRole
func (i InlineInterpretedTextRole) NodeType() NodeType { return i.Type }

// String satisfies the Stringer interface
func (i InlineInterpretedTextRole) String() string { return fmt.Sprintf("%#v", i) }

// MarshalJSON satisfies the Marshaler interface.
func (i InlineInterpretedTextRole) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type          string `json:"type"`
		Text          string `json:"text"`
		Length        int    `json:"length"`
		Line          int    `json:"line,omitempty"`
		StartPosition int    `json:"startPosition,omitempty"`
	}{
		Type:          nodeTypes[i.Type],
		Text:          i.Text,
		Length:        i.Length,
		Line:          i.Line,
		StartPosition: i.StartPosition,
	})
}

// BlockQuoteNode contains a parsed blockquote Node. Any nodes that are children of the blockquote are contained in NodeList.
type BlockQuoteNode struct {
	Type          NodeType `json:"type"`
	Line          int      `json:"line,omitempty"`
	StartPosition int      `json:"startPosition,omitempty"`
	// NodeList contains Nodes parsed as children of the BlockQuoteNode.
	NodeList `json:"nodeList"`
}

func NewEmptyBlockQuote(i *tok.Item) *BlockQuoteNode {
	bq := &BlockQuoteNode{
		Type:          NodeBlockQuote,
		Line:          i.Line,
		StartPosition: i.StartPosition,
	}
	return bq
}

func NewBlockQuote(i *tok.Item) *BlockQuoteNode {
	bq := &BlockQuoteNode{
		Type:          NodeBlockQuote,
		Line:          i.Line,
		StartPosition: i.StartPosition,
	}
	bq.NodeList.Append(NewParagraphWithNodeText(i))
	return bq
}

// NodeType returns the Node type of the BlockQuoteNode.
func (b BlockQuoteNode) NodeType() NodeType { return b.Type }

// String satisfies the Stringer interface
func (b BlockQuoteNode) String() string { return fmt.Sprintf("%#v", b) }

// MarshalJSON satisfies the Marshaler interface.
func (b BlockQuoteNode) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString("{")
	buffer.WriteString(fmt.Sprintf("\"type\": %q,", b.Type.String()))
	buffer.WriteString(fmt.Sprintf("\"line\": %d,", b.Line))
	buffer.WriteString(fmt.Sprintf("\"startPosition\": %d,", b.StartPosition))
	n, err := json.Marshal(b.NodeList)
	if err != nil {
		return nil, err
	}
	if string(n) == "null" {
		n = []byte{'[', ' ', ']'}
	}
	buffer.WriteString(fmt.Sprintf("\"nodeList\": %s", string(n)))
	buffer.WriteString("}")
	return buffer.Bytes(), nil
}

// SystemMessages contains system messages if present
type SystemMessagesNode struct {
	Type     NodeType          `json:"type"`
	NodeList `json:"nodeList"` // NodeList contains a list of system messages generated while parsing
}

func NewSystemMessagesNode() *SystemMessagesNode { return &SystemMessagesNode{Type: NodeSystemMessages} }

// NodeType returns the Node type of the SystemMessagesNode.
func (s SystemMessagesNode) NodeType() NodeType { return s.Type }

// String satisfies the Stringer interface
func (s SystemMessagesNode) String() string { return fmt.Sprintf("%#v", s) }

// MarshalJSON satisfies the Marshaler interface.
func (s SystemMessagesNode) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString("{")
	buffer.WriteString(fmt.Sprintf("\"type\": %q,", s.Type.String()))
	b, err := json.Marshal(s.NodeList)
	if err != nil {
		return nil, err
	}
	if string(b) == "null" {
		b = []byte{'[', ' ', ']'}
	}
	buffer.WriteString(fmt.Sprintf("\"nodeList\": %s", string(b)))
	buffer.WriteString("}")
	return buffer.Bytes(), nil
}

// SystemMessageNode are messages generated by the parser. System messages are leveled by severity and can be one of either
// Warning, Error, Info, and Severe.
type SystemMessageNode struct {
	Type NodeType `json:"type"`

	// The line containing the problem resulted in the system message
	Line int `json:"line,omitempty"`

	// The start line of the message block
	StartLine int `json:"startLine,omitempty"`

	// The end line of the message block
	EndLine int `json:"endLine,omitempty"`

	// The character position of the start of the problem that resulted in a system message
	StartPosition int `json:"StartPosition,omitempty"`

	// The type of parser message that generated the systemMessage.
	MessageType string `json:"messageType"`

	// Severity is the level of importance of the message. It can be one of either info, warning, error, and severe.
	Severity string `json:"severity"`

	// NodeList contains children Nodes of the systemMessage. Typically containing the first list item as a NodeParagraph
	// which contains the message, and a NodeLiteralBlock which contains the input data causing the systemMessage to be
	// generated.
	NodeList `json:"nodeList"`
}

func NewSystemMessage(pm *messages.ParserMessage, line int) *SystemMessageNode {
	sm := &SystemMessageNode{
		Type:        NodeSystemMessage,
		MessageType: pm.Type.String(),
		Severity:    pm.Level(),
		Line:        line,
	}
	msg := NewText(&tok.Item{
		Text:   pm.Message(),
		Length: len(pm.Message()),
	})
	sm.Append(msg)
	return sm
}

// NodeType returns the Node type of the SystemMessageNode.
func (s SystemMessageNode) NodeType() NodeType { return s.Type }

// String satisfies the Stringer interface
func (s SystemMessageNode) String() string { return fmt.Sprintf("%#v", s) }

// MarshalJSON satisfies the Marshaler interface.
func (s SystemMessageNode) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString("{")
	buffer.WriteString(fmt.Sprintf("\"type\": %q,", s.MessageType))
	buffer.WriteString(fmt.Sprintf("\"severity\": %q,", s.Severity))
	if s.Line > 0 {
		buffer.WriteString(fmt.Sprintf("\"line\": %d,", s.Line))
	}
	if s.StartLine > 0 {
		buffer.WriteString(fmt.Sprintf("\"startLine\": %d,", s.StartLine))
	}
	if s.EndLine > 0 {
		buffer.WriteString(fmt.Sprintf("\"endLine\": %d,", s.EndLine))
	}
	if s.StartPosition > 0 {
		buffer.WriteString(fmt.Sprintf("\"startPosition\": %d,", s.StartPosition))
	}
	b, err := json.Marshal(s.NodeList)
	if err != nil {
		return nil, err
	}
	if string(b) == "null" {
		b = []byte{'[', ' ', ']'}
	}
	buffer.WriteString(fmt.Sprintf("\"nodeList\": %s ", string(b)))
	buffer.WriteString("}")
	return buffer.Bytes(), nil
}

// LiteralBlockNode is a parsed literal block element.
type LiteralBlockNode struct {
	Type          NodeType `json:"type"`
	Text          string   `json:"text"`
	Length        int      `json:"length"`
	Line          int      `json:"line,omitempty"`
	StartPosition int      `json:"startPosition,omitempty"`
}

func NewLiteralBlock(i *tok.Item) *LiteralBlockNode {
	return &LiteralBlockNode{
		Type:          NodeLiteralBlock,
		Text:          i.Text,
		Length:        i.Length,
		Line:          i.Line,
		StartPosition: i.StartPosition,
	}
}

// NodeType returns the Node type of LiteralBlockNode.
func (l LiteralBlockNode) NodeType() NodeType { return l.Type }

// String satisfies the Stringer interface
func (l LiteralBlockNode) String() string { return fmt.Sprintf("%#v", l) }

// MarshalJSON satisfies the Marshaler interface.
func (l LiteralBlockNode) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type          string `json:"type"`
		Text          string `json:"text"`
		Length        int    `json:"length"`
		Line          int    `json:"line,omitempty"`
		StartPosition int    `json:"startPosition,omitempty"`
	}{
		Type:          nodeTypes[l.Type],
		Text:          l.Text,
		Length:        l.Length,
		Line:          l.Line,
		StartPosition: l.StartPosition,
	})
}

// TransitionNode is a parsed transition element. Transition elements are very similar to AdornmentNodes.
type TransitionNode struct {
	Type          NodeType `json:"type"`
	Text          string   `json:"text"`
	Length        int      `json:"length"`
	Line          int      `json:"line,omitempty"`
	StartPosition int      `json:"startPosition,omitempty"`
}

func NewTransition(i *tok.Item) *TransitionNode {
	return &TransitionNode{
		Type:          NodeTransition,
		Text:          i.Text,
		Length:        i.Length,
		Line:          i.Line,
		StartPosition: i.StartPosition,
	}
}

// NodeType returns the Node type of the TransitionNode.
func (t TransitionNode) NodeType() NodeType { return t.Type }

// String satisfies the Stringer interface
func (t TransitionNode) String() string { return fmt.Sprintf("%#v", t) }

// MarshalJSON satisfies the Marshaler interface.
func (t TransitionNode) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type          string `json:"type"`
		Text          string `json:"text"`
		Length        int    `json:"length"`
		Line          int    `json:"line,omitempty"`
		StartPosition int    `json:"startPosition,omitempty"`
	}{
		Type:          nodeTypes[t.Type],
		Text:          t.Text,
		Length:        t.Length,
		Line:          t.Line,
		StartPosition: t.StartPosition,
	})
}

// CommentNode is a parsed comment element. Comment elements do not appear as visible elements in document transformations.
type CommentNode struct {
	Type          NodeType `json:"type"`
	Text          string   `json:"text,omitempty"`
	Length        int      `json:"length,omitempty"`
	Line          int      `json:"line,omitempty"`
	StartPosition int      `json:"startPosition,omitempty"`
}

func NewComment(i *tok.Item) *CommentNode {
	return &CommentNode{
		Type:          NodeComment,
		Text:          i.Text,
		Length:        i.Length,
		Line:          i.Line,
		StartPosition: i.StartPosition,
	}
}

// NodeType returns the Node type of the CommentNode.
func (c CommentNode) NodeType() NodeType { return c.Type }

// String satisfies the Stringer interface
func (c CommentNode) String() string { return fmt.Sprintf("%#v", c) }

// MarshalJSON satisfies the Marshaler interface.
func (c CommentNode) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type          string `json:"type"`
		Text          string `json:"text,omitempty"`
		Length        int    `json:"length,omitempty"`
		Line          int    `json:"line,omitempty"`
		StartPosition int    `json:"startPosition,omitempty"`
	}{
		Type:          nodeTypes[c.Type],
		Text:          c.Text,
		Length:        c.Length,
		Line:          c.Line,
		StartPosition: c.StartPosition,
	})
}

// BulletListNode defines a bullet list element.
type BulletListNode struct {
	Type     NodeType `json:"type"`
	Bullet   string   `json:"bullet"`
	NodeList `json:"nodeList"`
}

// NewEnumListNode initializes a new BulletListNode.
func NewBulletListNode(i *tok.Item) *BulletListNode {
	return &BulletListNode{
		Type:   NodeBulletList,
		Bullet: i.Text,
	}
}

// NodeType returns the type of Node for the bullet list.
func (b BulletListNode) NodeType() NodeType { return b.Type }

// String satisfies the Stringer interface
func (b BulletListNode) String() string { return fmt.Sprintf("%#v", b) }

// MarshalJSON satisfies the Marshaler interface.
func (b BulletListNode) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString("{")
	buffer.WriteString(fmt.Sprintf("\"type\": %q,", b.Type.String()))
	buffer.WriteString(fmt.Sprintf("\"bullet\": %q,", b.Bullet))
	n, err := json.Marshal(b.NodeList)
	if err != nil {
		return nil, err
	}
	if string(n) == "null" {
		n = []byte{'[', ' ', ']'}
	}
	buffer.WriteString(fmt.Sprintf("\"nodeList\": %s", string(n)))
	buffer.WriteString("}")
	return buffer.Bytes(), nil
}

// BulletListItemNode defines a Bullet List Item element.
type BulletListItemNode struct {
	Type     NodeType `json:"type"`
	NodeList `json:"nodeList"`
}

// NewBulletListNode initializes a new EnumListNode.
func NewBulletListItemNode(i *tok.Item) *BulletListItemNode {
	return &BulletListItemNode{Type: NodeBulletListItem}
}

// NodeType returns the type of Node for the bullet list item.
func (b BulletListItemNode) NodeType() NodeType { return b.Type }

// String satisfies the Stringer interface
func (b BulletListItemNode) String() string { return fmt.Sprintf("%#v", b) }

// MarshalJSON satisfies the Marshaler interface.
func (b BulletListItemNode) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString("{")
	buffer.WriteString(fmt.Sprintf("\"type\": %q,", b.Type.String()))
	n, err := json.Marshal(b.NodeList)
	if err != nil {
		return nil, err
	}
	if string(n) == "null" {
		n = []byte{'[', ' ', ']'}
	}
	buffer.WriteString(fmt.Sprintf("\"nodeList\": %s", string(n)))
	buffer.WriteString("}")
	return buffer.Bytes(), nil
}

// EnumListNode defines an enumerated list element.
type EnumListNode struct {
	Type     NodeType      `json:"type"`
	EnumType EnumListType  `json:"enumType"`
	Affix    EnumAffixType `json:"affix"`
	NodeList `json:"nodeList"`
}

// NewEnumListNode initializes a new EnumListNode.
func NewEnumListNode(enumList *tok.Item, affix *tok.Item) *EnumListNode {
	var enType EnumListType
	switch enumList.Type {
	case tok.EnumListArabic:
		enType = enumListArabic
	}

	var afType EnumAffixType
	switch affix.Text {
	case ".":
		afType = enumAffixPeriod
		// case "(":
		// afType = enumAffixParenthesisSurround
		// case ")":
		// afType = enumAffixParenthesisRight
	}

	return &EnumListNode{
		Type:     NodeEnumList,
		EnumType: enType,
		Affix:    afType,
	}
}

// NodeType returns the Node type of the EnumListNode.
func (e EnumListNode) NodeType() NodeType { return e.Type }

// String satisfies the Stringer interface
func (e EnumListNode) String() string { return fmt.Sprintf("%#v", e) }

// MarshalJSON satisfies the Marshaler interface.
func (e EnumListNode) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString("{")
	buffer.WriteString(fmt.Sprintf("\"type\": %q,", e.Type.String()))
	buffer.WriteString(fmt.Sprintf("\"enumType\": %q,", e.EnumType))
	buffer.WriteString(fmt.Sprintf("\"affix\": %q,", e.Affix))
	b, err := json.Marshal(e.NodeList)
	if err != nil {
		return nil, err
	}
	if string(b) == "null" {
		b = []byte{'[', ' ', ']'}
	}
	buffer.WriteString(fmt.Sprintf("\"nodeList\": %s", string(b)))
	buffer.WriteString("}")
	return buffer.Bytes(), nil
}

// DefinitionListNode defines a definition list element.
type DefinitionListNode struct {
	Type     NodeType `json:"type"`
	NodeList `json:"nodeList"`
}

func NewDefinitionList(i *tok.Item) *DefinitionListNode {
	return &DefinitionListNode{Type: NodeDefinitionList}
}

// NodeType returns the Node type of DefinitionListNode.
func (d DefinitionListNode) NodeType() NodeType { return d.Type }

// String satisfies the Stringer interface
func (d DefinitionListNode) String() string { return fmt.Sprintf("%#v", d) }

// MarshalJSON satisfies the Marshaler interface.
func (d DefinitionListNode) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString("{")
	buffer.WriteString(fmt.Sprintf("\"type\": %q,", d.Type.String()))
	b, err := json.Marshal(d.NodeList)
	if err != nil {
		return nil, err
	}
	if string(b) == "null" {
		b = []byte{'[', ' ', ']'}
	}
	buffer.WriteString(fmt.Sprintf("\"nodeList\": %s", string(b)))
	buffer.WriteString("}")
	return buffer.Bytes(), nil
}

// DefinitionListItemNode defines a definition list item element.
type DefinitionListItemNode struct {
	Type       NodeType            `json:"type"`
	Term       *DefinitionTermNode `json:"term"`
	Definition *DefinitionNode     `json:"definition"`
}

func NewDefinitionListItem(defTerm *tok.Item, def *tok.Item) *DefinitionListItemNode {
	n := &DefinitionListItemNode{Type: NodeDefinitionListItem}
	ndt := &DefinitionTermNode{
		Type:          NodeDefinitionTerm,
		Text:          defTerm.Text,
		Length:        defTerm.Length,
		StartPosition: defTerm.StartPosition,
		Line:          defTerm.Line,
	}
	nd := &DefinitionNode{Type: NodeDefinition}
	n.Term = ndt
	n.Definition = nd
	return n
}

// NodeType returns the Node type of DefinitionListItemNode.
func (d DefinitionListItemNode) NodeType() NodeType { return d.Type }

// String satisfies the Stringer interface
func (d DefinitionListItemNode) String() string { return fmt.Sprintf("%#v", d) }

// MarshalJSON satisfies the Marshaler interface.
func (d DefinitionListItemNode) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type       string              `json:"type"`
		Term       *DefinitionTermNode `json:"term"`
		Definition *DefinitionNode     `json:"definition"`
	}{
		Type:       nodeTypes[d.Type],
		Term:       d.Term,
		Definition: d.Definition,
	})
}

// DefinitionTermNode defines a definition list term element.
type DefinitionTermNode struct {
	Type          NodeType `json:"type"`
	Text          string   `json:"text"`
	Length        int      `json:"length"`
	Line          int      `json:"line,omitempty"`
	StartPosition int      `json:"startPosition,omitempty"`
}

// NodeType returns the Node type of DefinitionTermNode.
func (d DefinitionTermNode) NodeType() NodeType { return d.Type }

// String satisfies the Stringer interface
func (d DefinitionTermNode) String() string { return fmt.Sprintf("%#v", d) }

// MarshalJSON satisfies the Marshaler interface.
func (d DefinitionTermNode) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type          string `json:"type"`
		Text          string `json:"text"`
		Length        int    `json:"length"`
		Line          int    `json:"line,omitempty"`
		StartPosition int    `json:"startPosition,omitempty"`
	}{
		Type:          nodeTypes[d.Type],
		Text:          d.Text,
		Length:        d.Length,
		Line:          d.Line,
		StartPosition: d.StartPosition,
	})
}

// DefinitionNode defines a difinition element.
type DefinitionNode struct {
	Type     NodeType `json:"type"`
	Line     int      `json:"line,omitempty"`
	NodeList `json:"nodeList"`
}

// NodeType returns the Node type of DefinitionNode.
func (d DefinitionNode) NodeType() NodeType { return d.Type }

// String satisfies the Stringer interface
func (d DefinitionNode) String() string { return fmt.Sprintf("%#v", d) }

// MarshalJSON satisfies the Marshaler interface.
func (d DefinitionNode) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString("{")
	buffer.WriteString(fmt.Sprintf("\"type\": %q,", d.Type.String()))
	if d.Line > 0 {
		buffer.WriteString(fmt.Sprintf("\"line\": %d,", d.Line))
	}
	b, err := json.Marshal(d.NodeList)
	if err != nil {
		return nil, err
	}
	if string(b) == "null" {
		b = []byte{'[', ' ', ']'}
	}
	buffer.WriteString(fmt.Sprintf("\"nodeList\": %s", string(b)))
	buffer.WriteString("}")
	return buffer.Bytes(), nil
}
