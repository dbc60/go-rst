// go-rst - A reStructuredText parser for Go
// 2014 (c) The go-rst Authors
// MIT Licensed. See LICENSE for details.

// To enable debug output when testing, use "go test -debug"

package parse

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"code.google.com/p/go.text/unicode/norm"
	"github.com/demizer/go-elog"
)

func init() { SetDebug() }

// SetDebug is typically called from the init() function in a test file. SetDebug parses debug flags
// passed to the test binary and also sets the template for logging output.
func SetDebug() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "Enable debug output.")
	flag.Parse()

	if debug {
		log.SetLevel(log.LEVEL_DEBUG)
	}

	log.SetTemplate("{{if .Date}}{{.Date}} {{end}}" +
		"{{if .Prefix}}{{.Prefix}} {{end}}" +
		"{{if .LogLabel}}{{.LogLabel}} {{end}}" +
		"{{if .FileName}}{{.FileName}}: {{end}}" +
		"{{if .FunctionName}}{{.FunctionName}}{{end}}" +
		"{{if .LineNumber}}#{{.LineNumber}}: {{end}}" +
		"{{if .Text}}{{.Text}}{{end}}")

	log.SetFlags(log.Lansi | log.LnoPrefix | log.LfunctionName |
		log.LlineNumber)
}

// Contains a single test with data loaded from test files in the testdata directory
type Test struct {
	path     string // The path including directory and basename
	data     string // The input data to be parsed
	itemData string // The expected lex items output in json
	nodeData string // The expected parse nodes in json
}

// expectNodes returns the expected parse_tree values from the tests as unmarshaled JSON. A panic
// occurs if there is an error unmarshaling the JSON data.
func (l Test) expectNodes() (nl []interface{}) {
	if err := json.Unmarshal([]byte(l.nodeData), &nl); err != nil {
		panic(fmt.Errorf("JSON error: ", err))
	}
	return
}

// expectItems unmarshals the expected lex_items into a silce of items. A panic occurs if there is
// an error decoding the JSON data.
func (l Test) expectItems() (lexItems []item) {
	if err := json.Unmarshal([]byte(l.itemData), &lexItems); err != nil {
		panic(fmt.Errorf("JSON error: ", err))
	}
	return
}

func LoadTest(path string) (test *Test) {
	inputData, err := ioutil.ReadFile("../testdata/" + path + ".rst")
	if err != nil {
		panic(err)
	}
	itemData, err := ioutil.ReadFile("../testdata/" + path + "_items.json")
	if err != nil {
		panic(err)
	}
	nodeData, err := ioutil.ReadFile("../testdata/" + path + "_nodes.json")
	if err != nil {
		panic(err)
	}
	return &Test{
		path:     path,
		data:     string(inputData[:len(inputData)-1]),
		itemData: string(itemData),
		nodeData: string(nodeData),
	}
}

type checkNode struct {
	t          *testing.T
	testPath   string
	pNodeName  string
	pFieldName string
	pFieldVal  interface{}
	pFieldType reflect.Type
	eFieldName string
	eFieldVal  interface{}
	eFieldType reflect.Type
	id         int
}

func (c *checkNode) error(args ...interface{}) {
	c.t.Error(args...)
}

func (c *checkNode) errorf(format string, args ...interface{}) {
	c.t.Errorf(format, args...)
}

func (c *checkNode) dError() {
	var got, exp string

	switch c.pFieldVal.(type) {
	case ID:
		got = c.pFieldVal.(ID).String()
		exp = strconv.Itoa(int(c.eFieldVal.(float64)))
	case NodeType:
		got = c.pFieldVal.(NodeType).String()
		exp = c.eFieldVal.(string)
	case StartPosition:
		got = c.pFieldVal.(StartPosition).String()
		exp = strconv.Itoa(int(c.eFieldVal.(float64)))
	case Line:
		got = c.pFieldVal.(Line).String()
		exp = strconv.Itoa(int(c.eFieldVal.(float64)))
	case string:
		got = c.pFieldVal.(string)
		exp = c.eFieldVal.(string)
	case int:
		got = strconv.Itoa(c.pFieldVal.(int))
		exp = strconv.Itoa(int(c.eFieldVal.(float64)))
	case rune:
		got = string(c.pFieldVal.(rune))
		exp = string(c.eFieldVal.(rune))
	}
	c.t.Errorf("(ID: %d) Got:\t%s = %q\n\t\tExpect: %s = %q\n\n", c.id, c.pFieldName, got,
		c.eFieldName, exp)
}

func (c *checkNode) updateState(eKey string, eVal interface{}, pVal reflect.Value) bool {
	// Expected parser metadata
	c.eFieldName = eKey
	c.eFieldVal = eVal
	c.eFieldType = reflect.TypeOf(c.eFieldVal)

	// Actual parsed metadata
	c.pNodeName = pVal.Type().Name()
	c.pFieldName = strings.ToUpper(string(c.eFieldName[0])) + c.eFieldName[1:]

	if c.pFieldName == "Id" {
		// Overide for uppercase ID
		c.pFieldName = "ID"
	}

	if !pVal.FieldByName(c.pFieldName).IsValid() {
		panic(fmt.Errorf("Missing field in parser output: %s.%s\n", c.pNodeName, c.pFieldName))
	}
	c.pFieldVal = pVal.FieldByName(c.pFieldName).Interface()
	c.pFieldType = pVal.FieldByName(c.pFieldName).Type()

	// Overline adornment nodes can be null
	if c.eFieldName == "overLine" && c.eFieldVal == nil {
		return false
	} else if c.eFieldVal == nil {
		c.dError()
		return false
	}

	return true
}

func (c *checkNode) checkFields(eNodes interface{}, pNode Node) {
	c.id = int(eNodes.(map[string]interface{})["id"].(float64))
	for eKey, eVal := range eNodes.(map[string]interface{}) {
		pVal := reflect.Indirect(reflect.ValueOf(pNode))
		if c.updateState(eKey, eVal, pVal) == false {
			continue
		}
		switch c.eFieldName {
		case "text":
			if norm.NFC.String(c.eFieldVal.(string)) != c.pFieldVal.(string) {
				c.dError()
			}
		case "type":
			if c.eFieldVal != c.pFieldVal.(NodeType).String() {
				c.dError()
			}
		case "id":
			if c.eFieldVal != float64(c.pFieldVal.(ID)) {
				c.dError()
			}
		case "level", "length":
			if c.eFieldVal != float64(c.pFieldVal.(int)) {
				c.dError()
			}
		case "line":
			if c.eFieldVal != float64(c.pFieldVal.(Line)) {
				c.dError()
			}
		case "startPosition":
			if c.eFieldVal != float64(c.pFieldVal.(StartPosition)) {
				c.dError()
			}
		case "indent", "overLine", "title", "underLine":
			c.checkFields(c.eFieldVal, c.pFieldVal.(Node))
		case "nodeList":
			for num, node := range c.eFieldVal.([]interface{}) {
				// Store and reset the parser value, otherwise a panic will occur on
				// the next iteration
				pFieldVal := c.pFieldVal
				c.checkFields(node, c.pFieldVal.(NodeList)[num])
				c.pFieldVal = pFieldVal
			}
		case "rune":
			if c.eFieldVal != string(c.pFieldVal.(rune)) {
				c.dError()
			}
		case "severity":
			if c.eFieldVal != c.pFieldVal.(systemMessageLevel).String() {
				c.dError()
			}
		default:
			panic(fmt.Errorf("%s is not implemented!", c.eFieldName))
		}
	}

}

func checkParseNodes(t *testing.T, eTree []interface{}, pNodes []Node, testPath string) {
	state := &checkNode{t: t, testPath: testPath}
	for eNum, eNode := range eTree {
		state.checkFields(eNode, pNodes[eNum])
	}
	return
}

var testSectionLevel = [...]SectionNode{
	SectionNode{
		OverLine:  &AdornmentNode{Rune: '='},
		UnderLine: &AdornmentNode{Rune: '='},
		Level:     1,
	},
	SectionNode{
		OverLine:  &AdornmentNode{Rune: '-'},
		UnderLine: &AdornmentNode{Rune: '-'},
		Level:     2,
	},
	SectionNode{
		UnderLine: &AdornmentNode{Rune: '~'},
		Level:     3,
	},
}

func TestSectionLevelsAdd(t *testing.T) {
	x := new(sectionLevels)
	v := &testSectionLevel[0]
	x.Add(v)
	if v.Level != 1 {
		t.Errorf("Improper level on first add, Got level: %d, expected: %d", v.Level, 1)
	}
}

func TestSectionLevelsFind(t *testing.T) {
	var p sectionLevels
	p.Add(&testSectionLevel[0])
	p.Add(&testSectionLevel[1])
	p.Add(&testSectionLevel[2])
	sec := p.FindByRune('-')
	if sec.Level == -1 {
		t.Errorf("Expect:\t%d\nGot:\t%t\n", 2, sec.Level)
	}
	if sec.Level != 2 {
		t.Errorf("Expect:\t%d\nGot:\t%d\n", 2, sec.Level)
	}
}

func TestSectionLevelsFindNoResult(t *testing.T) {
	var p sectionLevels
	p.Add(&testSectionLevel[0])
	sec := p.FindByRune('-')
	if sec != nil {
		t.Error("Expect no return value!")
	}
}

func TestSectionLevelsLevelEmpty(t *testing.T) {
	var p sectionLevels
	lvl := p.Level()
	expect := 0
	if lvl != expect {
		t.Errorf("Empty sectionLevels should return \"%d\"!\nExpect:\t%d\nGot:\t%d\n",
			expect, expect, lvl)
	}

}

func TestSectionLevelsLevel(t *testing.T) {
	var p sectionLevels
	p.Add(&testSectionLevel[0])
	p.Add(&testSectionLevel[1])
	p.Add(&testSectionLevel[2])
	lvl := p.Level()
	if lvl != 3 {
		t.Errorf("Level() returned incorrect level!\nExpect:\n\n\t%d\nGot:\n\n\t%d\n", 3, lvl)
	}
}

func parseTest(t *testing.T, test *Test) (tree *Tree) {
	log.Debugf("Test path: %s\n", test.path)
	log.Debugf("Test Input:\n-----------\n%s\n----------\n", test.data)
	tree, _ = Parse(test.path, test.data)
	return
}

var treeNextTests = []struct {
	name     string
	input    string
	nextNum  int   // Number of times to call Tree.next(). Value starts at 1.
	back3Tok *item // The item to expect at Tree.token[zed-3]
	back2Tok *item // The item to expect at Tree.token[zed-2]
	back1Tok *item // The item to expect at Tree.token[zed-1]
	zedToken *item // The item to expect at Tree.token[zed]
}{
	{
		name:    "Next no input",
		input:   "",
		nextNum: 1,
		// zedToken should be nil
	},
	{
		name:     "Single next from start",
		input:    "Test\n=====\n\nParagraph.",
		nextNum:  1,
		zedToken: &item{Type: itemTitle, Text: "Test"},
	},
	{
		name:     "Double next",
		input:    "Test\n=====\n\nParagraph.",
		nextNum:  2,
		back1Tok: &item{Type: itemTitle, Text: "Test"},
		zedToken: &item{Type: itemSectionAdornment, Text: "====="},
	},
	{
		name:     "Tripple next",
		input:    "Test\n=====\n\nParagraph.",
		nextNum:  3,
		back2Tok: &item{Type: itemTitle, Text: "Test"},
		back1Tok: &item{Type: itemSectionAdornment, Text: "====="},
		zedToken: &item{Type: itemBlankLine, Text: "\n"},
	},
	{
		name:     "Quadrupple next",
		input:    "Test\n=====\n\nParagraph.",
		nextNum:  4,
		back3Tok: &item{Type: itemTitle, Text: "Test"},
		back2Tok: &item{Type: itemSectionAdornment, Text: "====="},
		back1Tok: &item{Type: itemBlankLine, Text: "\n"},
		zedToken: &item{Type: itemParagraph, Text: "Paragraph."},
	},
	{
		name:     "Two next() on one line of input",
		input:    "Test",
		nextNum:  2,
		back1Tok: &item{Type: itemParagraph, Text: "Test"},
		zedToken: &item{Type: itemEOF, Text: ""},
	},
}

func TestTreeNext(t *testing.T) {
	for _, tt := range treeNextTests {
		tree := New(tt.name, tt.input)
		tree.lex = lex(tt.name, tt.input)
		if tt.nextNum > 4 {
			panic("nextNum cannot be greater than 4")
		}
		for i := 0; i < tt.nextNum; i++ {
			tree.next()
		}
		if tree.token[zed] == nil && tt.zedToken == nil {
			continue
		}
		if tt.nextNum > 0 {
			if tree.token[zed].Type != tt.zedToken.Type {
				t.Errorf("Test: %s\n\t Got: token[zed].Type = %s, Expect: %s\n\n",
					tree.Name, tree.token[zed].Type, tt.zedToken.Type)
			}
			if tree.token[zed].Text != tt.zedToken.Text {
				t.Errorf("Test: %s\n\t Got: token[zed].Text = %s, Expect: %q\n\n",
					tree.Name, tree.token[zed].Text, tt.zedToken.Text)
			}
		}
		if tt.nextNum > 1 {
			if tree.token[zed-1].Type != tt.back1Tok.Type {
				t.Errorf("Test: %s\n\t Got: token[zed-1].Type = %s, Expect: %s\n\n",
					tree.Name, tree.token[zed-1].Type, tt.back1Tok.Type)
			}
			if tree.token[zed-1].Text != tt.back1Tok.Text {
				t.Errorf("Test: %s\n\t Got: token[zed-1].Text = %q, Expect: %q\n\n",
					tree.Name, tree.token[zed-1].Text, tt.back1Tok.Text)
			}
		}
		if tt.nextNum > 2 {
			if tree.token[zed-2].Type != tt.back2Tok.Type {
				t.Errorf("Test: %s\n\t Got: token[zed-2].Type = %s, Expect: %s\n\n",
					tree.Name, tree.token[zed-2].Type, tt.back2Tok.Type)
			}
			if tree.token[zed-2].Text != tt.back2Tok.Text {
				t.Errorf("Test: %s\n\t Got: token[zed-2].Text = %s, Expect: %q\n\n",
					tree.Name, tree.token[zed-2].Text, tt.back2Tok.Text)
			}
		}
		if tt.nextNum > 3 {
			if tree.token[zed-3].Type != tt.back3Tok.Type {
				t.Errorf("Test: %s\n\t Got: token[zed-3].Type = %s, Expect: %s\n\n",
					tree.Name, tree.token[zed-3].Type, tt.back3Tok.Type)
			}
			if tree.token[zed-3].Text != tt.back3Tok.Text {
				t.Errorf("Test: %s\n\t Got: token[zed-3].Text = %s, Expect: %q\n\n",
					tree.Name, tree.token[zed-3].Text, tt.back3Tok.Text)
			}
		}
		// Calculate the number of open backup tokens and make sure they are
		// empty.
		for j := 0; j < 4-tt.nextNum; j++ {
			if tree.token[j] != nil {
				t.Errorf("Test: %s\n\t Got: token[%d] = %#+v, Expect: nil\n\n",
					tree.Name, j, tree.token[j])
			}
		}
		// make sure the peek positions are blank
		if tree.token[zed+1] != nil {
			t.Errorf("Test: %s\n\t Got: token[zed+1] = %#+v, Expect: nil\n\n",
				tree.Name, tree.token[zed+1])
		} else if tree.token[zed+2] != nil {
			t.Errorf("Test: %s\n\t Got: token[zed+2] = %#+v, Expect: nil\n\n",
				tree.Name, tree.token[zed+2])
		} else if tree.token[zed+3] != nil {
			t.Errorf("Test: %s\n\t Got: token[zed+3] = %#+v, Expect: nil\n\n",
				tree.Name, tree.token[zed+3])
		}
		// spd.Dump(tree.token)
	}
}

var treePeekTests = []struct {
	name     string
	input    string
	nextNum  int  // Number of times to call Tree.next() before peek
	peekNum  int  // position argument to Tree.peek()
	peek1Tok item // The item to expect at Tree.token[zed+1]
	peek2Tok item // The item to expect at Tree.token[zed+2]
	peek3Tok item // The item to expect at Tree.token[zed+3]
}{
	{
		name:    "Peek from starting position",
		input:   "Test\n=====\n\nParagraph.",
		nextNum: 0, peekNum: 1,
		peek1Tok: item{Type: itemTitle, Text: "Test"},
	},
	{
		name:    "Peek from starting position two times",
		input:   "Test\n=====\n\nParagraph.",
		nextNum: 0, peekNum: 2,
		peek1Tok: item{Type: itemTitle, Text: "Test"},
		peek2Tok: item{Type: itemSectionAdornment, Text: "====="},
	},
	{
		name:    "Peek three positions",
		input:   "Test\n=====\n\nParagraph.",
		nextNum: 0, peekNum: 3,
		peek1Tok: item{Type: itemTitle, Text: "Test"},
		peek2Tok: item{Type: itemSectionAdornment, Text: "====="},
		peek3Tok: item{Type: itemBlankLine, Text: "\n"},
	},
	{
		name:    "Next 2 positions, Peek 3 positions",
		input:   "Test\n=====\n\nParagraph.\nTest 2\n=====\n\nParagraph 2.",
		nextNum: 2, peekNum: 3,
		peek1Tok: item{Type: itemBlankLine, Text: "\n"},
		peek2Tok: item{Type: itemParagraph, Text: "Paragraph."},
		peek3Tok: item{Type: itemTitle, Text: "Test 2"},
	},
}

func TestTreePeek(t *testing.T) {
	for _, tt := range treePeekTests {
		tree := New(tt.name, tt.input)
		tree.lex = lex(tt.name, tt.input)
		for i := 0; i < tt.nextNum; i++ {
			tree.next()
		}
		tree.peek(tt.peekNum)
		// spd.Dump(tree.token)
		if tree.token[zed+1].Type != tt.peek1Tok.Type {
			t.Errorf("Test: %s\n\t Got: token[zed+1].Type = %s, Expect: %s\n\n",
				tree.Name, tree.token[zed+1].Type, tt.peek1Tok.Type)
		}
		if tree.token[zed+1].Text != tt.peek1Tok.Text {
			t.Errorf("Test: %s\n\t Got: token[zed+1].Text = %s, Expect: %q\n\n",
				tree.Name, tree.token[zed+1].Text, tt.peek1Tok.Text)
		}
		if tt.peekNum > 1 {
			if tree.token[zed+2].Type != tt.peek2Tok.Type {
				t.Errorf("Test: %s\n\t Got: token[zed+2].Type = %s, Expect: %s\n\n",
					tree.Name, tree.token[zed+2].Type, tt.peek2Tok.Type)
			}
			if tree.token[zed+2].Text != tt.peek2Tok.Text {
				t.Errorf("Test: %s\n\t Got: token[zed+2].Text = %s, Expect: %q\n\n",
					tree.Name, tree.token[zed+2].Text, tt.peek2Tok.Text)
			}
		}
		if tt.peekNum > 2 {
			if tree.token[zed+3].Type != tt.peek3Tok.Type {
				t.Errorf("Test: %s\n\t Got: token[zed+3].Type = %s, Expect: %s\n\n",
					tree.Name, tree.token[zed+3].Type, tt.peek3Tok.Type)
			}
			if tree.token[zed+3].Text != tt.peek3Tok.Text {
				t.Errorf("Test: %s\n\t Got: token[zed+3].Text = %s, Expect: %q\n\n",
					tree.Name, tree.token[zed+3].Text, tt.peek3Tok.Text)
			}
		}
	}
}

func TestParseSection001(t *testing.T) {
	testPath := "test_section/001_title_paragraph"
	test := LoadTest(testPath)
	pTree := parseTest(t, test)
	eNodes := test.expectNodes()
	checkParseNodes(t, eNodes, *pTree.Nodes, testPath)
}

func TestParseSection002(t *testing.T) {
	testPath := "test_section/002_paragraph_nbl"
	test := LoadTest(testPath)
	pTree := parseTest(t, test)
	eNodes := test.expectNodes()
	checkParseNodes(t, eNodes, *pTree.Nodes, testPath)
}

func TestParseSection003(t *testing.T) {
	testPath := "test_section/003_para_head_para"
	test := LoadTest(testPath)
	pTree := parseTest(t, test)
	eNodes := test.expectNodes()
	checkParseNodes(t, eNodes, *pTree.Nodes, testPath)
}

func TestParseSection004(t *testing.T) {
	testPath := "test_section/004_section_level_test"
	test := LoadTest(testPath)
	pTree := parseTest(t, test)
	eNodes := test.expectNodes()
	checkParseNodes(t, eNodes, *pTree.Nodes, testPath)
}

func TestParseSection005(t *testing.T) {
	testPath := "test_section/005_unexpected_titles"
	test := LoadTest(testPath)
	pTree := parseTest(t, test)
	eNodes := test.expectNodes()
	checkParseNodes(t, eNodes, *pTree.Nodes, testPath)
}

func TestParseSection006(t *testing.T) {
	testPath := "test_section/006_short_underline"
	test := LoadTest(testPath)
	pTree := parseTest(t, test)
	eNodes := test.expectNodes()
	checkParseNodes(t, eNodes, *pTree.Nodes, testPath)
}

func TestParseSection007(t *testing.T) {
	testPath := "test_section/007_title_combining_chars"
	test := LoadTest(testPath)
	pTree := parseTest(t, test)
	eNodes := test.expectNodes()
	checkParseNodes(t, eNodes, *pTree.Nodes, testPath)
}

func TestParseSection008(t *testing.T) {
	testPath := "test_section/008_title_overline"
	test := LoadTest(testPath)
	pTree := parseTest(t, test)
	eNodes := test.expectNodes()
	checkParseNodes(t, eNodes, *pTree.Nodes, testPath)
}

func TestParseSection009(t *testing.T) {
	testPath := "test_section/009_inset_title_with_overline"
	test := LoadTest(testPath)
	pTree := parseTest(t, test)
	eNodes := test.expectNodes()
	checkParseNodes(t, eNodes, *pTree.Nodes, testPath)
}

func TestParseSection010(t *testing.T) {
	testPath := "test_section/010_inset_title_missing_underline"
	test := LoadTest(testPath)
	pTree := parseTest(t, test)
	eNodes := test.expectNodes()
	checkParseNodes(t, eNodes, *pTree.Nodes, testPath)
}

func TestParseSection011(t *testing.T) {
	testPath := "test_section/011_inset_title_missing_underline_with_blankline"
	test := LoadTest(testPath)
	pTree := parseTest(t, test)
	eNodes := test.expectNodes()
	checkParseNodes(t, eNodes, *pTree.Nodes, testPath)
}

func TestParseSection012(t *testing.T) {
	testPath := "test_section/012_inset_title_missing_underline_and_para"
	test := LoadTest(testPath)
	pTree := parseTest(t, test)
	eNodes := test.expectNodes()
	checkParseNodes(t, eNodes, *pTree.Nodes, testPath)
}

func TestParseSection013(t *testing.T) {
	testPath := "test_section/013_title_too_long"
	test := LoadTest(testPath)
	pTree := parseTest(t, test)
	eNodes := test.expectNodes()
	checkParseNodes(t, eNodes, *pTree.Nodes, testPath)
}

func TestParseSection014(t *testing.T) {
	testPath := "test_section/014_inset_title_mismatched_underline"
	test := LoadTest(testPath)
	pTree := parseTest(t, test)
	eNodes := test.expectNodes()
	checkParseNodes(t, eNodes, *pTree.Nodes, testPath)
}

func TestParseSection015(t *testing.T) {
	testPath := "test_section/015_missing_titles_with_blankline"
	test := LoadTest(testPath)
	pTree := parseTest(t, test)
	eNodes := test.expectNodes()
	checkParseNodes(t, eNodes, *pTree.Nodes, testPath)
}

func TestParseSection016(t *testing.T) {
	testPath := "test_section/016_missing_titles_with_nbl"
	test := LoadTest(testPath)
	pTree := parseTest(t, test)
	eNodes := test.expectNodes()
	// spd.Dump(pTree.Nodes)
	checkParseNodes(t, eNodes, *pTree.Nodes, testPath)
}
