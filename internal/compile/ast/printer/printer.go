package printer

import (
	"fmt"
	"io"
	"strings"

	"codeberg.org/rileyq/usagi/internal/compile/ast"
	"codeberg.org/rileyq/usagi/internal/compile/token"
)

func Fprint(w io.Writer, node ast.Node) error {
	return fprint(w, node, 0)
}

func fprint(w io.Writer, node ast.Node, depth int) error {
	const pad = "  "

	var err error
	switch node := node.(type) {
	case *ast.Module:
		for _, decl := range node.Decls {
			err = fprint(w, decl, depth)
			if err != nil {
				return err
			}
			_, err = io.WriteString(w, "\n")
			if err != nil {
				return err
			}
		}
		return nil
	case *ast.Binding:
		var decls []string
		if node.Mode.Export() {
			decls = append(decls, "export")
		}
		if node.Mode.Const() {
			decls = append(decls, "const")
		}
		switch node.Token {
		case token.Const:
		case token.Trait:
			if node.Value.(*ast.TraitExpr).Closed {
				decls = append(decls, "trait(closed)")
				break
			}
			fallthrough
		default:
			decls = append(decls, node.Token.String())
		}
		decls = append(decls, node.Name.Name)
		_, err = io.WriteString(w, strings.Repeat(pad, depth))
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, strings.Join(decls, " "))
		if err != nil {
			return err
		}
		switch node.Token {
		case token.Const, token.Let:
			if node.Type != nil {
				_, err = io.WriteString(w, ": ")
				if err != nil {
					return err
				}
				err = fprint(w, node.Type, depth)
				if err != nil {
					return err
				}
			}
			if node.Value != nil {
				_, err = io.WriteString(w, " = ")
				if err != nil {
					return err
				}
				err = fprint(w, node.Value, depth)
				if err != nil {
					return err
				}
			}
			_, err = io.WriteString(w, ";")
			if err != nil {
				return err
			}
		case token.Func:
			expr := node.Value.(*ast.FuncExpr)
			err = funcBody(w, expr, depth)
			if err != nil {
				return err
			}
			if expr.Body == nil {
				_, err = io.WriteString(w, ";")
				if err != nil {
					return err
				}
			}
		case token.Struct:
			_, err = io.WriteString(w, " ")
			if err != nil {
				return err
			}
			err = structMembers(w, node.Value.(*ast.StructExpr), depth)
			if err != nil {
				return err
			}
			_, err = io.WriteString(w, ";")
			if err != nil {
				return err
			}
		case token.Trait:
			err = traitBody(w, node.Value.(*ast.TraitExpr), depth)
			if err != nil {
				return err
			}
		}
		_, err = io.WriteString(w, "\n")
		if err != nil {
			return err
		}
		return nil
	case *ast.ImplDecl:
		curPad := strings.Repeat(pad, depth)
		_, err = io.WriteString(w, curPad)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, "impl ")
		if err != nil {
			return err
		}
		err = fprint(w, node.Type, depth)
		if err != nil {
			return err
		}
		if len(node.Traits) > 0 {
			err = list(w, node.Traits, depth)
			if err != nil {
				return err
			}
		}
		err = listWithDelim(w, node.Definitions, depth+1, " {\n", curPad+"}\n", "")
		if err != nil {
			return err
		}
		return nil
	case *ast.ExprStmt:
		_, err = io.WriteString(w, strings.Repeat(pad, depth))
		if err != nil {
			return err
		}
		err = fprint(w, node.X, depth)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, ";\n")
		if err != nil {
			return err
		}
		return nil
	case *ast.DeclStmt:
		err = fprint(w, node.X, depth)
		if err != nil {
			return err
		}
		return nil
	case *ast.FuncExpr:
		_, err = io.WriteString(w, "func")
		if err != nil {
			return err
		}
		err = funcBody(w, node, depth)
		if err != nil {
			return err
		}
		return nil
	case *ast.TraitExpr:
		_, err = io.WriteString(w, "trait")
		if err != nil {
			return err
		}
		if node.Closed {
			_, err = io.WriteString(w, "(closed)")
			if err != nil {
				return err
			}
		}
		err = traitBody(w, node, depth)
		if err != nil {
			return err
		}
		return nil
	case *ast.Param:
		err = fprint(w, node.Name, depth)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, ": ")
		if err != nil {
			return err
		}
		err = fprint(w, node.Type, depth)
		if err != nil {
			return err
		}
		return nil
	case *ast.BlockExpr:
		_, err = io.WriteString(w, "{\n")
		if err != nil {
			return err
		}
		for _, stmt := range node.List {
			err = fprint(w, stmt, depth+1)
			if err != nil {
				return err
			}
		}
		_, err = io.WriteString(w, strings.Repeat(pad, depth))
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, "}")
		if err != nil {
			return err
		}
		return nil
	case *ast.BinaryExpr:
		err = fprint(w, node.Left, depth)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, " "+node.Op.String()+" ")
		if err != nil {
			return err
		}
		err = fprint(w, node.Right, depth)
		if err != nil {
			return err
		}
		return nil
	case *ast.UnaryExpr:
		_, err = io.WriteString(w, node.Op.String())
		if err != nil {
			return err
		}
		err = fprint(w, node.Base, depth)
		if err != nil {
			return err
		}
		return nil
	case *ast.MemberExpr:
		err = fprint(w, node.Base, depth)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, ".")
		if err != nil {
			return err
		}
		err = fprint(w, node.Member, depth)
		if err != nil {
			return err
		}
		return nil
	case *ast.CallExpr:
		err = fprint(w, node.Base, depth)
		if err != nil {
			return err
		}
		err = list(w, node.Args, depth)
		if err != nil {
			return err
		}
		return nil
	case *ast.IndexExpr:
		err = fprint(w, node.Base, depth)
		if err != nil {
			return err
		}
		err = listWithDelim(w, node.Indices, depth, "[", "]", ", ")
		if err != nil {
			return err
		}
		return nil
	case *ast.ReturnExpr:
		_, err = io.WriteString(w, "return ")
		if err != nil {
			return err
		}
		err = fprint(w, node.Value, depth)
		if err != nil {
			return err
		}
		return nil
	case *ast.ExistentialExpr:
		_, err = io.WriteString(w, "forSome ")
		if err != nil {
			return err
		}
		err = fprint(w, node.Base, depth)
		if err != nil {
			return err
		}
		return nil
	case *ast.SliceExpr:
		_, err = io.WriteString(w, "[]")
		if err != nil {
			return err
		}
		err = fprint(w, node.Base, depth)
		if err != nil {
			return err
		}
		return nil
	case *ast.NamedArg:
		err = fprint(w, node.Name, depth)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, ": ")
		if err != nil {
			return err
		}
		err = fprint(w, node.Value, depth)
		if err != nil {
			return err
		}
		return nil
	case *ast.Field:
		err = fprint(w, node.Name, depth)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, ": ")
		if err != nil {
			return err
		}
		err = fprint(w, node.Type, depth)
		if err != nil {
			return err
		}
		return nil
	case *ast.Identifier:
		_, err = io.WriteString(w, node.Name)
		return err
	case *ast.Literal:
		_, err = io.WriteString(w, node.Value)
		return err
	default:
		return fmt.Errorf("printer: Fprint unimplemented for %T", node)
	}
}

func funcBody(w io.Writer, node *ast.FuncExpr, depth int) error {
	err := list(w, node.Params, depth)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, " ")
	if err != nil {
		return err
	}
	err = fprint(w, node.ReturnType, depth)
	if err != nil {
		return err
	}
	if node.Body != nil {
		_, err = io.WriteString(w, " ")
		if err != nil {
			return err
		}
		err = fprint(w, node.Body, depth)
		if err != nil {
			return err
		}
	}
	return nil
}

func structMembers(w io.Writer, node *ast.StructExpr, depth int) error {
	_, err := io.WriteString(w, "(\n")
	if err != nil {
		return err
	}
	pad := strings.Repeat("  ", depth)
	innerPad := strings.Repeat("  ", depth+1)
	for _, m := range node.Members {
		_, err = io.WriteString(w, innerPad)
		if err != nil {
			return err
		}
		err = fprint(w, m, depth+1)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, ",\n")
		if err != nil {
			return err
		}
	}
	_, err = io.WriteString(w, pad)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, ")")
	if err != nil {
		return err
	}
	return nil
}

func traitBody(w io.Writer, node *ast.TraitExpr, depth int) error {
	var err error
	if len(node.Traits) > 0 {
		err = list(w, node.Traits, depth)
		if err != nil {
			return err
		}
	}

	err = listWithDelim(w, node.Members, depth+1, " {\n", "}", "")
	if err != nil {
		return err
	}

	return nil
}

func list[T ast.Node](w io.Writer, exprs []T, depth int) error {
	return listWithDelim(w, exprs, depth, "(", ")", ", ")
}

func listWithDelim[T ast.Node](w io.Writer, exprs []T, depth int, open, close, sep string) error {
	_, err := io.WriteString(w, open)
	if err != nil {
		return err
	}
	for i, expr := range exprs {
		err = fprint(w, expr, depth)
		if err != nil {
			return err
		}
		if i < len(exprs)-1 {
			_, err = io.WriteString(w, sep)
			if err != nil {
				return err
			}
		}
	}
	_, err = io.WriteString(w, close)
	if err != nil {
		return err
	}
	return nil
}
