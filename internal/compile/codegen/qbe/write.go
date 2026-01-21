package qbe

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

func (m *Module) WriteTo(w io.Writer) (int64, error) {
	var total int64
	for _, def := range m.Definitions {
		n64, err := def.WriteTo(w)
		total += n64
		if err != nil {
			return total, err
		}
		n, err := io.WriteString(w, "\n\n")
		total += int64(n)
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func (d *Data) WriteTo(w io.Writer) (int64, error) {
	var total int64
	n64, err := d.Linkage.WriteTo(w)
	total += n64
	if err != nil {
		return total, err
	}
	n, err := io.WriteString(w, " data ")
	total += int64(n)
	if err != nil {
		return total, err
	}
	n, err = w.Write([]byte{' '})
	total += int64(n)
	if err != nil {
		return total, err
	}
	n64, err = d.Name.WriteTo(w)
	total += n64
	if err != nil {
		return total, err
	}
	n, err = w.Write([]byte{' ', '=', ' ', '{', ' '})
	total += int64(n)
	if err != nil {
		return total, err
	}
	for i, item := range d.Items {
		n64, err = item.WriteTo(w)
		total += n64
		if err != nil {
			return total, err
		}
		if i < len(d.Items)-1 {
			n, err = w.Write([]byte{',', ' '})
			total += int64(n)
			if err != nil {
				return total, err
			}
		}
	}
	n, err = w.Write([]byte{' ', '}'})
	total += int64(n)
	if err != nil {
		return total, err
	}
	return total, nil
}

func (lit StringLiteral) WriteTo(w io.Writer) (int64, error) {
	n, err := io.WriteString(w, "b "+strconv.Quote(string(lit))+", b 0")
	return int64(n), err
}

func (f *Function) WriteTo(w io.Writer) (int64, error) {
	var total int64
	n64, err := f.Linkage.WriteTo(w)
	total += n64
	if err != nil {
		return total, err
	}
	n, err := io.WriteString(w, " function ")
	total += int64(n)
	if err != nil {
		return total, err
	}
	n64, err = f.ReturnType.WriteTo(w)
	total += n64
	if err != nil {
		return total, err
	}
	n, err = w.Write([]byte{' ', '$'})
	total += int64(n)
	if err != nil {
		return total, err
	}
	n, err = io.WriteString(w, f.Name)
	total += int64(n)
	if err != nil {
		return total, err
	}
	n, err = w.Write([]byte{'('})
	total += int64(n)
	if err != nil {
		return total, err
	}
	for _, param := range f.Params {
		n64, err = param.WriteTo(w)
		total += n64
		if err != nil {
			return total, err
		}
	}
	n, err = w.Write([]byte{')', ' ', '{', '\n'})
	total += int64(n)
	if err != nil {
		return total, err
	}
	for _, block := range f.Blocks {
		n64, err = block.WriteTo(w)
		total += n64
		if err != nil {
			return total, err
		}
		n, err = w.Write([]byte{'\n'})
		total += int64(n)
		if err != nil {
			return total, err
		}
	}
	n, err = w.Write([]byte{'}'})
	total += int64(n)
	if err != nil {
		return total, err
	}
	return total, nil
}

func (l Linkage) WriteTo(w io.Writer) (int64, error) {
	var linkage []string
	if l.Export() {
		linkage = append(linkage, "export")
	}
	if l.Thread() {
		linkage = append(linkage, "thread")
	}
	n, err := io.WriteString(w, strings.Join(linkage, " "))
	return int64(n), err
}

func (b *Block) WriteTo(w io.Writer) (int64, error) {
	var total int64
	n, err := io.WriteString(w, "@"+b.Name)
	total += int64(n)
	if err != nil {
		return total, err
	}
	n, err = w.Write([]byte{'\n'})
	total += int64(n)
	if err != nil {
		return total, err
	}
	for i, inst := range b.Instructions {
		n64, err := inst.WriteTo(w)
		total += n64
		if err != nil {
			return total, err
		}
		if i < len(b.Instructions)-1 {
			n, err = w.Write([]byte{'\n'})
			total += int64(n)
			if err != nil {
				return total, err
			}
		}
	}
	return total, nil
}

func (p *Param) WriteTo(w io.Writer) (int64, error) {
	var total int64
	n64, err := p.Type.WriteTo(w)
	total += n64
	if err != nil {
		return total, err
	}
	n, err := w.Write([]byte{' '})
	total += int64(n)
	if err != nil {
		return total, err
	}
	n64, err = p.Ident.WriteTo(w)
	total += n64
	if err != nil {
		return total, err
	}
	return total, nil
}

func (t *Temporary) WriteTo(w io.Writer) (int64, error) {
	n, err := io.WriteString(w, "%"+t.name)
	return int64(n), err
}

func (t SimpleType) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write([]byte(t))
	return int64(n), err
}

func (g Global) WriteTo(w io.Writer) (int64, error) {
	n, err := io.WriteString(w, "$"+g.name)
	return int64(n), err
}

func (c Constant) WriteTo(w io.Writer) (int64, error) {
	n, err := fmt.Fprintf(w, "%d", int64(c))
	return int64(n), err
}

func (r *Ret) WriteTo(w io.Writer) (int64, error) {
	var total int64
	n, err := w.Write([]byte("\tret"))
	total += int64(n)
	if err != nil {
		return total, err
	}
	if r.Value != nil {
		n, err = w.Write([]byte{' '})
		total += int64(n)
		if err != nil {
			return total, err
		}
		n64, err := r.Value.WriteTo(w)
		total += n64
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func (c *Call) WriteTo(w io.Writer) (int64, error) {
	var total int64
	n, err := w.Write([]byte{'\t'})
	total += int64(n)
	if err != nil {
		return total, err
	}
	if c.Out != nil {
		n64, err := c.Out.WriteTo(w)
		total += n64
		if err != nil {
			return total, err
		}
		n, err = w.Write([]byte{' ', '='})
		total += int64(n)
		if err != nil {
			return total, err
		}
		n64, err = c.Type.WriteTo(w)
		total += int64(n)
		if err != nil {
			return total, err
		}
		n, err = w.Write([]byte{' '})
		total += int64(n)
		if err != nil {
			return total, err
		}
	}
	n, err = io.WriteString(w, "call ")
	total += int64(n)
	if err != nil {
		return total, err
	}
	n64, err := c.Base.WriteTo(w)
	total += n64
	if err != nil {
		return total, err
	}
	n, err = w.Write([]byte{'('})
	total += int64(n)
	if err != nil {
		return total, err
	}
	for i, arg := range c.Args {
		n64, err = arg.Type().WriteTo(w)
		total += n64
		if err != nil {
			return total, err
		}
		n, err = w.Write([]byte{' '})
		total += int64(n)
		if err != nil {
			return total, err
		}
		n64, err = arg.WriteTo(w)
		total += n64
		if err != nil {
			return total, err
		}
		if i < len(c.Args)-1 {
			n, err = w.Write([]byte{',', ' '})
			total += int64(n)
			if err != nil {
				return total, err
			}
		}
	}
	n, err = w.Write([]byte{')'})
	total += int64(n)
	if err != nil {
		return total, err
	}
	return total, nil
}

func (t *threeAddress) WriteTo(w io.Writer) (int64, error) {
	var total int64
	n, err := w.Write([]byte{'\t'})
	total += int64(n)
	if err != nil {
		return total, err
	}
	n64, err := t.out.WriteTo(w)
	total += n64
	if err != nil {
		return total, err
	}
	n, err = w.Write([]byte{' ', '='})
	total += int64(n)
	if err != nil {
		return total, err
	}
	n64, err = t.typ.WriteTo(w)
	total += n64
	if err != nil {
		return total, err
	}
	n, err = w.Write([]byte{' '})
	total += int64(n)
	if err != nil {
		return total, err
	}
	n, err = io.WriteString(w, t.name)
	total += int64(n)
	if err != nil {
		return total, err
	}
	n, err = w.Write([]byte{' '})
	total += int64(n)
	if err != nil {
		return total, err
	}
	n64, err = t.a.WriteTo(w)
	total += n64
	if err != nil {
		return total, err
	}
	return total, nil
}
