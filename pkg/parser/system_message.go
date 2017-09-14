package parser

import (
	doc "github.com/demizer/go-rst/pkg/document"
	mes "github.com/demizer/go-rst/pkg/messages"
	tok "github.com/demizer/go-rst/pkg/token"
)

func (p *Parser) systemMessageSection(s *doc.SystemMessageNode, err *mes.ParserMessage) {
	switch err.Type {
	case mes.SectionWarningOverlineTooShortForTitle:
		overline := p.buf[p.index-2]
		title := p.buf[p.index-1]
		// For title with only overline, combine two tokens and insert into buffer
		err.LiteralText = overline.Text + "\n" + p.token.Text
		// For title with overline and underline, combine 3 tokens and insert into buffer
		if p.index-2 >= 0 && overline != nil && overline.Type == tok.SectionAdornment {
			err.LiteralText = overline.Text + "\n" + title.Text + "\n" + p.token.Text
		}
		err.Line = overline.Line
		err.StartPosition = overline.StartPosition
	case mes.SectionWarningUnexpectedTitleOverlineOrTransition:
		err.LiteralText = p.peekBackTo(tok.SectionAdornment).Text + "\n" + p.peekBackTo(tok.Title).Text + "\n" + p.token.Text
	case mes.SectionWarningUnderlineTooShortForTitle:
		err.LiteralText = p.buf[p.index-1].Text + "\n" + p.buf[p.index].Text
	case mes.SectionWarningShortOverline, mes.SectionErrorOverlineUnderlineMismatch:
		var indent string
		backIndex := p.index - 2
		if p.peekBack(2).Type == tok.Space {
			backIndex = p.index - 3
			indent = p.buf[p.index-2].Text
		}
		overLine := p.buf[backIndex].Text
		title := p.buf[p.index-1].Text
		underLine := p.token.Text
		newLine := "\n"
		err.LiteralText = overLine + newLine + indent + title + newLine + underLine
	case mes.SectionWarningShortUnderline, mes.SectionErrorUnexpectedSectionTitle:
		backIndex := p.index - 1
		if p.peekBack(1).Type == tok.Space {
			backIndex = p.index - 2
		}
		err.LiteralText = p.buf[backIndex].Text + "\n" + p.token.Text
		s.Line = p.buf[backIndex].Line
	case mes.SectionErrorInvalidSectionOrTransitionMarker:
		err.LiteralText = p.buf[p.index-1].Text + "\n" + p.token.Text
	case mes.SectionErrorIncompleteSectionTitle,
		mes.SectionErrorMissingMatchingUnderlineForOverline:
		err.LiteralText = p.buf[p.index-2].Text + "\n" + p.buf[p.index-1].Text + p.token.Text
	case mes.SectionErrorUnexpectedSectionTitleOrTransition:
		err.LiteralText = p.token.Text
	case mes.SectionErrorTitleLevelInconsistent:
		if p.peekBack(2).Type == tok.SectionAdornment {
			err.LiteralText = p.buf[p.index-2].Text + "\n" + p.buf[p.index-1].Text + "\n" + p.token.Text
			break
		}
		err.LiteralText = p.buf[p.index-1].Text + "\n" + p.token.Text
	}
}

func (p *Parser) systemMessageInlineMarkup(s *doc.SystemMessageNode, err *mes.ParserMessage) *doc.LiteralBlockNode {
	switch err.Type {
	case mes.InlineMarkupWarningExplicitMarkupWithUnIndent:
		s.Line = p.peek(1).Line
	}
	return nil
}

// systemMessage generates a Node based on the passed mes.ParserMessage. The generated message is returned as a
// SystemMessageNode.
func (p *Parser) systemMessage(err mes.MessageType) (ok bool) {
	nm := mes.NewParserMessage(err)
	s := doc.NewSystemMessage(nm, p.token.Line)
	p.systemMessageSection(s, nm)
	p.systemMessageInlineMarkup(s, nm)
	return false
}
