package messages

import "strings"

// MessageType implements messages generated by the parser. Parser messages are leveled using systemMessageLevels.
type MessageType int

const (
	ParserMessageNil MessageType = iota
	SectionWarningOverlineTooShortForTitle
	SectionWarningUnexpectedTitleOverlineOrTransition
	SectionWarningUnderlineTooShortForTitle
	SectionWarningShortOverline
	SectionWarningShortUnderline
	SectionErrorInvalidSectionOrTransitionMarker
	SectionErrorUnexpectedSectionTitle
	SectionErrorUnexpectedSectionTitleOrTransition
	SectionErrorIncompleteSectionTitle
	SectionErrorMissingMatchingUnderlineForOverline
	SectionErrorOverlineUnderlineMismatch
	SectionErrorTitleLevelInconsistent
	InlineMarkupWarningExplicitMarkupWithUnIndent
)

var messageTypes = [...]string{
	"ParserMessageNil",
	"SectionWarningOverlineTooShortForTitle",
	"SectionWarningUnexpectedTitleOverlineOrTransition",
	"SectionWarningUnderlineTooShortForTitle",
	"SectionWarningShortOverline",
	"SectionWarningShortUnderline",
	"SectionErrorInvalidSectionOrTransitionMarker",
	"SectionErrorUnexpectedSectionTitle",
	"SectionErrorUnexpectedSectionTitleOrTransition",
	"SectionErrorIncompleteSectionTitle",
	"SectionErrorMissingMatchingUnderlineForOverline",
	"SectionErrorOverlineUnderlineMismatch",
	"SectionErrorTitleLevelInconsistent",
	"InlineMarkupWarningExplicitMarkupWithUnIndent",
}

// String implements Stringer and returns the MessageType as a string. The returned string is the MessageType name, not
// the message itself.
func (m MessageType) String() string { return messageTypes[m] }

// message returns the message of the MessageType as a string.
func (m MessageType) message() (s string) {
	switch m {
	case SectionWarningOverlineTooShortForTitle:
		s = "Possible incomplete section title.\nTreating the overline as ordinary text because it's so short."
	case SectionWarningUnexpectedTitleOverlineOrTransition:
		s = "Unexpected possible title overline or transition.\nTreating it as ordinary text because it's so short."
	case SectionWarningUnderlineTooShortForTitle:
		s = "Possible title underline, too short for the title.\nTreating it as ordinary text because it's so short."
	case SectionWarningShortOverline:
		s = "Title overline too short."
	case SectionWarningShortUnderline:
		s = "Title underline too short."
	case SectionErrorInvalidSectionOrTransitionMarker:
		s = "Invalid section title or transition marker."
	case SectionErrorUnexpectedSectionTitle:
		s = "Unexpected section title."
	case SectionErrorUnexpectedSectionTitleOrTransition:
		s = "Unexpected section title or transition."
	case SectionErrorIncompleteSectionTitle:
		s = "Incomplete section title."
	case SectionErrorMissingMatchingUnderlineForOverline:
		s = "Missing matching underline for section title overline."
	case SectionErrorOverlineUnderlineMismatch:
		s = "Title overline & underline mismatch."
	case SectionErrorTitleLevelInconsistent:
		s = "Title level inconsistent."
	case InlineMarkupWarningExplicitMarkupWithUnIndent:
		s = "Explicit markup ends without a blank line; unexpected unindent."
	}
	return
}

// level returns the MessageType level.
func (m MessageType) level() (s string) {
	if strings.Contains(m.String(), "Warning") {
		s = "WARNING"
	} else {
		s = "ERROR"
	}
	return
}

// IsSectionMessage returns true if the MessageType m is a section message type.
func IsSectionMessage(m MessageType) bool { return strings.Contains(m.String(), "Section") }

// IsInlineMarkupMessage returns true if the MessageType m is a inline markup message type.
func IsInlineMarkupMessage(m MessageType) bool { return strings.Contains(m.String(), "InlineMarkup") }
