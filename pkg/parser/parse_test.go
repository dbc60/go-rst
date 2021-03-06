package parser

import (
	"io/ioutil"
	"testing"

	"github.com/demizer/go-rst/pkg/testutil"

	doc "github.com/demizer/go-rst/pkg/document"
)

func nodeListToInterface(v *doc.NodeList) []interface{} {
	v2 := []doc.Node(*v)
	s := make([]interface{}, len(v2))
	for i, j := range v2 {
		s[i] = j
	}
	return s
}

// checkParseNodes compares the expected parser output (*_nodes.json) against the actual parser output using the jd library.
func checkParseNodes(t *testing.T, expectNodes string, p *Parser, testPath string) {
	pJson, err := doc.JsonRenderer(testutil.LoggerConfig, p.Messages, p.Nodes).Bytes()
	if err != nil {
		t.Errorf("Error Marshalling JSON: %s", err.Error())
		return
	}

	//
	// Json diff output has a syntax: https://github.com/josephburnett/jd#diff-language
	//
	o, err := testutil.JsonDiff(expectNodes, string(pJson))
	if err != nil {
		t.Errorf("Error diffing JSON: %s", err.Error())
		return
	}

	// There should be no output from the diff
	if len(o) != 0 {
		testutil.Log("\nFAIL: parsed nodes do not match expected nodes!")
		testutil.Log("\n[Parsed Nodes JSON]\n\n")
		testutil.Log(string(pJson))
		testutil.Log("\n\n[JSON DIFF]\n\n")
		testutil.Log(o)
		t.FailNow()
	}
}

func LoadParserTest(t *testing.T, path string) (test *testutil.Test) {
	iDPath := path + ".rst"
	inputData, err := ioutil.ReadFile(iDPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(inputData) == 0 {
		t.Fatalf("\"%s\" is empty!", iDPath)
	}
	nDPath := path + "-nodes.json"
	eNodes, err := ioutil.ReadFile(nDPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(eNodes) == 0 {
		t.Fatalf("\"%s\" is empty!", nDPath)
	}
	return &testutil.Test{
		Path:            path,
		Data:            string(inputData[:len(inputData)-1]),
		ExpectParseData: string(eNodes),
	}
}

// parseTest initiates the parser and parses a test using test.data is input.
func parseTest(t *testing.T, test *testutil.Test) *Parser {
	p, err := NewParser(test.Path, test.Data, testutil.LoggerConfig)
	if err != nil {
		panic(err)
	}
	p.Msgr("test path", "path", test.Path)
	p.Msgr("test input", "input", test.Data)
	p.Parse()
	return p
}
