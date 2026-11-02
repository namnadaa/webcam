package i18n

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// sourceTexts collects the string literals passed to i18n.T and i18n.Tf in the module.
func sourceTexts(t *testing.T) map[string]string {
	t.Helper()
	texts := map[string]string{}

	err := filepath.WalkDir("../..", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && path != "../.." {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "i18n" || (sel.Sel.Name != "T" && sel.Sel.Name != "Tf") {
				return true
			}
			lit, ok := call.Args[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				t.Errorf("%s: i18n.%s needs a string literal to be checked", fset.Position(call.Pos()), sel.Sel.Name)
				return true
			}
			s, err := strconv.Unquote(lit.Value)
			if err != nil {
				t.Fatal(err)
			}
			texts[s] = fset.Position(call.Pos()).String()
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return texts
}

func TestRussianCoversSource(t *testing.T) {
	texts := sourceTexts(t)
	if len(texts) == 0 {
		t.Fatal("no i18n.T calls found")
	}
	for s, pos := range texts {
		if _, ok := ru[s]; !ok {
			t.Errorf("%s: no Russian translation for %q", pos, s)
		}
	}
	for s := range ru {
		if _, ok := texts[s]; !ok {
			t.Errorf("unused Russian translation for %q", s)
		}
	}
}

var verb = regexp.MustCompile(`%[-+# 0]*[0-9.*]*[a-zA-Z%]`)

func TestRussianKeepsFormatVerbs(t *testing.T) {
	for en, tr := range ru {
		if a, b := verb.FindAllString(en, -1), verb.FindAllString(tr, -1); !reflect.DeepEqual(a, b) {
			t.Errorf("format verbs differ:\n  en %q %v\n  ru %q %v", en, a, tr, b)
		}
	}
}

func TestT(t *testing.T) {
	defer Set(English)

	Set(English)
	if got := Tf("Noise threshold: %.1f DN", 2.0); got != "Noise threshold: 2.0 DN" {
		t.Errorf("English Tf = %q", got)
	}

	Set(Russian)
	if got := T("no such text"); got != "no such text" {
		t.Errorf("untranslated text = %q, want it unchanged", got)
	}
	if got := T("Noise threshold: %.1f DN"); got == "Noise threshold: %.1f DN" {
		t.Error("Russian translation not used")
	}
}

func TestParseLang(t *testing.T) {
	tests := []struct {
		input string
		want  Lang
		valid bool
	}{
		{"", English, true},
		{"1", English, true},
		{" 2 \r", Russian, true}, // Windows console line ending
		{"d", English, false},
		{"3", English, false},
		{"12", English, false},
	}
	for _, tt := range tests {
		if got, valid := parseLang(tt.input); got != tt.want || valid != tt.valid {
			t.Errorf("parseLang(%q) = %v %v, want %v %v", tt.input, got, valid, tt.want, tt.valid)
		}
	}
}

func TestReadLineKeepsRest(t *testing.T) {
	r := strings.NewReader("d\n2\n")

	if line, ok := readLine(r); line != "d" || !ok {
		t.Errorf("first line = %q %v, want \"d\" true", line, ok)
	}
	if line, ok := readLine(r); line != "2" || !ok {
		t.Errorf("second line = %q %v, want \"2\" true", line, ok)
	}
	if _, ok := readLine(r); ok {
		t.Error("readLine at the end of input returned ok")
	}
}
