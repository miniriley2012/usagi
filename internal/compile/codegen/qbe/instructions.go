package qbe

type Ret struct{ Value Value }

func NewRet(value Value) *Ret { return &Ret{value} }

type Call struct {
	Out  Value
	Type Type
	Base Value
	Args []Value
}

func NewCall(out Value, typ Type, base Value, args []Value) *Call { return &Call{out, typ, base, args} }

type threeAddress struct {
	name string
	out  Value
	typ  Type
	a, b Value
}

type Add struct{ threeAddress }

func NewAdd(out Value, typ Type, a, b Value) *Add {
	return &Add{threeAddress{"add", out, typ, a, b}}
}
