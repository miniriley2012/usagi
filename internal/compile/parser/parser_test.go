package parser

import (
	"bytes"
	"testing"

	"codeberg.org/rileyq/usagi/internal/compile/ast/printer"
)

const src = `
const std = @import("std");

struct TwoInts (
	a: i32,
	b: i32,
);

const Type = trait(closed) {};

trait(closed) Type {}

trait Drop {
	func drop(self: Self) void;
}

// Equivalent to
//   trait Linear {}
//   impl Linear(!Drop);
trait Linear(!Drop) {}

impl TwoInts(Drop) {
	func drop(self: TwoInts) void {}
}

func genericAdd(x: forSome Integer, y: @TypeOf(x)) @TypeOf(x) {
	return x + y;
}

func ArrayList(T: forSome Type) forSome Type {
	struct ArrayList(data: []T, size: usize, capacity: usize);
	impl ArrayList {
		const empty = ArrayList(data: []T(), size: 0, capacity: 0);

		func append(self: Self, item: T) void {
			self.data[self.size] = item;
			self.size = self.size + 1;
		}
	}
	return ArrayList;
}

// Adds two i32 values
const add: func(arg: TwoInts) i32 = func(arg: TwoInts) i32 {
	return arg.a + arg.b;
};

func main() void {
	std.print(add(TwoInts(a: 1, b: 2)));
}
`

func TestParser(t *testing.T) {
	p := NewFromReader(bytes.NewReader([]byte(src)))
	module, err := p.Parse("main")
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	err = printer.Fprint(t.Output(), module)
	if err != nil {
		t.Fatal(err)
	}
}
