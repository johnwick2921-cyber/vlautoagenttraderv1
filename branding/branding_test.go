package branding

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

func TestCanonicalDisplayNames(t *testing.T) {
	if ProductName() != "VL Intelligent" || PersonaName() != "VL" {
		t.Fatalf("noncanonical display names: product=%q persona=%q", ProductName(), PersonaName())
	}
}
func bannerReadsName(source []byte) bool {
	file, err := parser.ParseFile(token.NewFileSet(), "main.go", source, 0)
	if err != nil {
		return false
	}
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "Info" {
			return true
		}
		receiver, ok := selector.X.(*ast.Ident)
		if !ok || receiver.Name != "logger" {
			return true
		}
		isBanner, readsName := false, false
		ast.Inspect(call, func(child ast.Node) bool {
			if value, ok := child.(*ast.BasicLit); ok && strings.Contains(value.Value, "AI-Powered Trading System") {
				isBanner = true
			}
			if value, ok := child.(*ast.CallExpr); ok {
				if getter, ok := value.Fun.(*ast.SelectorExpr); ok {
					if pkg, ok := getter.X.(*ast.Ident); ok && pkg.Name == "branding" && getter.Sel.Name == "ProductName" {
						readsName = true
					}
				}
			}
			return true
		})
		if isBanner && readsName {
			found = true
		}
		return true
	})
	return found
}
func TestBootBannerReadsSharedName(t *testing.T) {
	source, err := os.ReadFile("../main.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bannerReadsName(source) {
		t.Fatal("boot banner does not read branding.ProductName()")
	}
	// A same-looking literal is still a contract failure; guard the production call site.
	literal := strings.Replace(string(source), "branding.ProductName()", `"VL Intelligent"`, 1)
	if literal == string(source) || bannerReadsName([]byte(literal)) {
		t.Fatal("literal banner mutation escaped the shared-source guard")
	}
}
