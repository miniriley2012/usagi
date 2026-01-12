package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
)

func main() {
	inputPath := os.Args[1]
	outputPath := os.Args[2]

	data, err := os.ReadFile(inputPath)
	if err != nil {
		panic(err)
	}

	var input Input
	err = json.Unmarshal(data, &input)
	if err != nil {
		panic(err)
	}

	output, err := fileToString(fileFromInput(&input))
	if err != nil {
		panic(err)
	}

	if outputPath == "-" {
		fmt.Print(output)
	} else {
		err = os.WriteFile(outputPath, []byte(output), 0o666)
		if err != nil {
			panic(err)
		}
	}
}

func fileFromInput(input *Input) *ast.File {
	dynamic := slices.Sorted(slices.Values(input.Dynamic))
	keywords := slices.Sorted(slices.Values(input.Keywords))
	fixedKeys := slices.Sorted(maps.Keys(input.Fixed))

	file := new(ast.File)
	file.Name = ast.NewIdent("token")

	file.Decls = append(file.Decls, &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent("Type"),
				Type: ast.NewIdent("int"),
			},
		},
	})

	tokenCount := len(dynamic) + len(keywords) + len(fixedKeys) + 1

	specs := make([]ast.Spec, 0, tokenCount)
	values := make([]ast.Expr, 0, tokenCount)

	specs = append(specs, &ast.ValueSpec{
		Names:  []*ast.Ident{ast.NewIdent("Invalid")},
		Type:   ast.NewIdent("Type"),
		Values: []ast.Expr{ast.NewIdent("iota")},
	})

	values = append(values, &ast.BasicLit{
		Kind:  token.STRING,
		Value: "\"<invalid>\"",
	})

	for _, name := range dynamic {
		specs = append(specs, &ast.ValueSpec{
			Names: []*ast.Ident{ast.NewIdent(exportName(name))},
		})
		values = append(values, &ast.BasicLit{
			Kind:  token.STRING,
			Value: strconv.Quote("<" + name + ">"),
		})
	}

	for _, name := range keywords {
		specs = append(specs, &ast.ValueSpec{
			Names: []*ast.Ident{ast.NewIdent(exportName(name))},
		})
		values = append(values, &ast.BasicLit{
			Kind:  token.STRING,
			Value: strconv.Quote(name),
		})
	}

	for _, name := range fixedKeys {
		specs = append(specs, &ast.ValueSpec{
			Names: []*ast.Ident{ast.NewIdent(exportName(name))},
		})
		values = append(values, &ast.BasicLit{
			Kind:  token.STRING,
			Value: strconv.Quote(input.Fixed[name]),
		})
	}

	file.Decls = append(file.Decls, &ast.GenDecl{
		Tok:    token.CONST,
		Lparen: 1,
		Specs:  specs,
		Rparen: 1,
	})

	file.Decls = append(file.Decls, &ast.FuncDecl{
		Recv: &ast.FieldList{
			List: []*ast.Field{{
				Names: []*ast.Ident{ast.NewIdent("t")},
				Type:  ast.NewIdent("Type"),
			}},
		},
		Name: ast.NewIdent("String"),
		Type: &ast.FuncType{
			Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("string")}}},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.IfStmt{
					Cond: &ast.BinaryExpr{
						X: &ast.BinaryExpr{
							X:  ast.NewIdent("t"),
							Op: token.LSS,
							Y:  &ast.BasicLit{Kind: token.INT, Value: "0"},
						},
						Op: token.LOR,
						Y: &ast.BinaryExpr{
							X:  ast.NewIdent("t"),
							Op: token.GTR,
							Y:  specs[len(specs)-1].(*ast.ValueSpec).Names[0],
						},
					},
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							&ast.AssignStmt{
								Lhs: []ast.Expr{ast.NewIdent("t")},
								Tok: token.ASSIGN,
								Rhs: []ast.Expr{ast.NewIdent("Invalid")},
							},
						},
					},
				},
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.IndexExpr{
							X:     ast.NewIdent("names"),
							Index: ast.NewIdent("t"),
						},
					},
				},
			},
		},
	})

	file.Decls = append(file.Decls, &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{&ast.ValueSpec{
			Names: []*ast.Ident{ast.NewIdent("names")},
			Values: []ast.Expr{&ast.CompositeLit{
				Type: &ast.ArrayType{Elt: ast.NewIdent("string")},
				Elts: values,
			}},
		}},
	})

	file.Decls = append(file.Decls, &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{&ast.TypeSpec{
			Name: ast.NewIdent("TrieNode"),
			Type: &ast.StructType{
				Fields: &ast.FieldList{
					List: []*ast.Field{
						{
							Names: []*ast.Ident{ast.NewIdent("Rune")},
							Type:  ast.NewIdent("rune"),
						},
						{
							Names: []*ast.Ident{ast.NewIdent("Type")},
							Type:  ast.NewIdent("Type"),
						},
						{
							Names: []*ast.Ident{ast.NewIdent("Children")},
							Type:  &ast.ArrayType{Elt: &ast.StarExpr{X: ast.NewIdent("TrieNode")}},
						},
					},
				},
			},
		}},
	})

	fixedTokens := maps.Clone(input.Fixed)
	for _, name := range keywords {
		fixedTokens[name] = name
		fixedKeys = append(fixedKeys, name)
	}
	slices.Sort(fixedKeys)
	fixedValues := slices.Sorted(maps.Values(fixedTokens))

	fixedTokensInverse := make(map[string]string, len(input.Fixed))
	for key, value := range fixedTokens {
		fixedTokensInverse[value] = key
	}

	root := &trieNode{}

	for _, key := range fixedValues {
		root.Insert(key, ast.NewIdent(exportName(fixedTokensInverse[key])))
	}

	file.Decls = append(file.Decls, &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{&ast.ValueSpec{
			Names:  []*ast.Ident{ast.NewIdent("Fixed")},
			Values: []ast.Expr{root.Expr(true)},
		}},
	})

	return file
}

func exportName(name string) string {
	return strings.ToTitle(name[:1]) + name[1:]
}

func fileToString(f *ast.File) (string, error) {
	var b strings.Builder
	b.WriteString("// Code generated by generate_tokens.go\n\n")
	err := format.Node(&b, token.NewFileSet(), f)
	return b.String(), err
}

type Input struct {
	Dynamic  []string          `json:"dynamic"`
	Keywords []string          `json:"keywords"`
	Fixed    map[string]string `json:"fixed"`
}

type trieNode struct {
	Rune     rune
	Type     ast.Expr
	Children []*trieNode
}

func (node *trieNode) Insert(key string, value ast.Expr) {
outer:
	for _, r := range key {
		for i := range node.Children {
			c := node.Children[i]
			if r == node.Rune {
				node = c
				continue outer
			}
		}
		node.Children = append(node.Children, &trieNode{Rune: r})
		node = node.Children[len(node.Children)-1]
	}
	node.Type = value
}

func (node *trieNode) Expr(withType bool) ast.Expr {
	var litType ast.Expr
	if withType {
		litType = ast.NewIdent("&TrieNode")
	}
	tokenType := node.Type
	if tokenType == nil {
		tokenType = ast.NewIdent("Invalid")
	}
	var children ast.Expr
	if len(node.Children) > 0 {
		elts := make([]ast.Expr, 0, len(node.Children))
		for _, c := range node.Children {
			elts = append(elts, c.Expr(false))
		}
		children = &ast.CompositeLit{
			Type: &ast.ArrayType{Elt: &ast.StarExpr{X: ast.NewIdent("TrieNode")}},
			Elts: elts,
		}
	} else {
		children = ast.NewIdent("nil")
	}
	return &ast.CompositeLit{
		Type: litType,
		Elts: []ast.Expr{
			&ast.BasicLit{
				Kind:  token.CHAR,
				Value: fmt.Sprintf("%q", node.Rune),
			},
			tokenType,
			children,
		},
	}
}
