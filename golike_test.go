package golike

import (
	"fmt"
	"reflect"
	"testing"
)

func copyAs[T any](t *testing.T, src any) T {
	t.Helper()
	var dst T
	if err := Copy(src, &dst); err != nil {
		t.Fatalf("Copy(%#v) returned unexpected error: %v", src, err)
	}
	return dst
}

func assertEqual(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func assertLenCap(t *testing.T, got any, wantLen, wantCap int) {
	t.Helper()
	v := reflect.ValueOf(got)
	if v.Len() != wantLen || v.Cap() != wantCap {
		t.Errorf("got len=%d cap=%d, want len=%d cap=%d", v.Len(), v.Cap(), wantLen, wantCap)
	}
}

func assertErr(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %q, got nil", want)
	}
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func assertPanics(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic, but function returned normally")
		}
	}()
	f()
}

type myInt int
type ids []int

type myErr struct{ msg string }

func (e myErr) Error() string { return e.msg }

func TestCopy_NonPointerTargetReturnsError(t *testing.T) {
	targets := []any{
		1,
		"str",
		[]int{1},
		[2]int{},
		map[string]int{},
		struct{}{},
		3.14,
	}
	for _, target := range targets {
		t.Run(fmt.Sprintf("%T", target), func(t *testing.T) {
			err := Copy(1, target)
			assertErr(t, err, fmt.Sprintf("expected pointer, got %T", target))
		})
	}
}

func TestCopy_Scalars(t *testing.T) {
	t.Run("int", func(t *testing.T) { assertEqual(t, copyAs[int](t, 42), 42) })
	t.Run("negative int", func(t *testing.T) { assertEqual(t, copyAs[int](t, -7), -7) })
	t.Run("int64", func(t *testing.T) { assertEqual(t, copyAs[int64](t, int64(1<<40)), int64(1<<40)) })
	t.Run("uint8", func(t *testing.T) { assertEqual(t, copyAs[uint8](t, uint8(255)), uint8(255)) })
	t.Run("float64", func(t *testing.T) { assertEqual(t, copyAs[float64](t, 3.14), 3.14) })
	t.Run("string", func(t *testing.T) { assertEqual(t, copyAs[string](t, "hello"), "hello") })
	t.Run("empty string", func(t *testing.T) { assertEqual(t, copyAs[string](t, ""), "") })
	t.Run("bool", func(t *testing.T) { assertEqual(t, copyAs[bool](t, true), true) })
	t.Run("named int to same named type", func(t *testing.T) {
		assertEqual(t, copyAs[myInt](t, myInt(5)), myInt(5))
	})
	t.Run("overwrites existing value", func(t *testing.T) {
		dst := 100
		if err := Copy(1, &dst); err != nil {
			t.Fatal(err)
		}
		assertEqual(t, dst, 1)
	})
}

func TestCopy_ScalarIntoInterface(t *testing.T) {
	assertEqual(t, copyAs[any](t, 5), any(5))
	assertEqual(t, copyAs[any](t, "s"), any("s"))
	assertEqual(t, copyAs[any](t, 2.5), any(2.5))
}

func TestCopy_ConcreteIntoNonEmptyInterface(t *testing.T) {
	var err error
	if e := Copy(myErr{"boom"}, &err); e != nil {
		t.Fatal(e)
	}
	if err == nil || err.Error() != "boom" {
		t.Errorf("got %v, want error 'boom'", err)
	}
}

func TestCopy_StructIsCopiedByValue(t *testing.T) {
	type S struct {
		A int
		B string
	}
	src := S{A: 1, B: "x"}
	dst := copyAs[S](t, src)
	assertEqual(t, dst, src)

	dst.A = 99
	if src.A != 1 {
		t.Errorf("mutating dst changed src: %+v", src)
	}
}

func TestCopy_MapIsShallow(t *testing.T) {
	src := map[string]int{"a": 1}
	dst := copyAs[map[string]int](t, src)
	assertEqual(t, dst, src)

	dst["a"] = 2
	if src["a"] != 2 {
		t.Error("expected map to share underlying storage (shallow copy)")
	}
}

func TestCopy_StructWithSliceFieldIsShallow(t *testing.T) {
	type S struct{ Items []int }
	src := S{Items: []int{1, 2, 3}}
	dst := copyAs[S](t, src)

	dst.Items[0] = 100
	if src.Items[0] != 100 {
		t.Error("expected slice field inside a struct to share backing array (shallow copy)")
	}
}

func TestCopy_PointerTarget(t *testing.T) {
	t.Run("allocates nil target pointer", func(t *testing.T) {
		var dst *int
		if err := Copy(7, &dst); err != nil {
			t.Fatal(err)
		}
		if dst == nil || *dst != 7 {
			t.Fatalf("got %v, want pointer to 7", dst)
		}
	})

	t.Run("replaces existing pointer instead of writing through it", func(t *testing.T) {
		x := 1
		dst := &x
		if err := Copy(7, &dst); err != nil {
			t.Fatal(err)
		}
		if *dst != 7 {
			t.Errorf("*dst = %d, want 7", *dst)
		}
		if x != 1 {
			t.Errorf("original pointee was modified: x = %d", x)
		}
		if dst == &x {
			t.Error("dst still points at the original variable")
		}
	})

	t.Run("double pointer target", func(t *testing.T) {
		var dst **int
		if err := Copy(7, &dst); err != nil {
			t.Fatal(err)
		}
		if dst == nil || *dst == nil || **dst != 7 {
			t.Fatalf("got %v, want **dst == 7", dst)
		}
	})

	t.Run("triple pointer target", func(t *testing.T) {
		var dst ***string
		if err := Copy("deep", &dst); err != nil {
			t.Fatal(err)
		}
		if dst == nil || *dst == nil || **dst == nil || ***dst != "deep" {
			t.Fatalf("unexpected result: %v", dst)
		}
	})
}

func TestCopy_PointerSource(t *testing.T) {
	t.Run("single pointer is dereferenced", func(t *testing.T) {
		x := 5
		assertEqual(t, copyAs[int](t, &x), 5)
	})

	t.Run("pointer to pointer is dereferenced", func(t *testing.T) {
		x := 5
		px := &x
		assertEqual(t, copyAs[int](t, &px), 5)
	})

	t.Run("pointer to pointer on both sides", func(t *testing.T) {
		x := 5
		px := &x
		dst := copyAs[**int](t, &px)
		if **dst != 5 {
			t.Errorf("**dst = %d, want 5", **dst)
		}
	})

	t.Run("result does not alias the source pointee", func(t *testing.T) {
		x := 5
		dst := copyAs[*int](t, &x)
		if dst == &x {
			t.Fatal("dst aliases src")
		}
		*dst = 6
		if x != 5 {
			t.Errorf("writing through dst changed src: x = %d", x)
		}
	})

	t.Run("pointer to struct", func(t *testing.T) {
		type S struct{ A int }
		src := &S{A: 3}
		assertEqual(t, copyAs[S](t, src), S{A: 3})
	})
}

func TestCopy_SliceToSlice(t *testing.T) {
	t.Run("ints", func(t *testing.T) {
		assertEqual(t, copyAs[[]int](t, []int{1, 2, 3}), []int{1, 2, 3})
	})

	t.Run("strings", func(t *testing.T) {
		assertEqual(t, copyAs[[]string](t, []string{"a", "b"}), []string{"a", "b"})
	})

	t.Run("structs", func(t *testing.T) {
		type S struct {
			A int
			B string
		}
		src := []S{{1, "x"}, {2, "y"}}
		assertEqual(t, copyAs[[]S](t, src), src)
	})

	t.Run("empty non-nil", func(t *testing.T) {
		got := copyAs[[]int](t, []int{})
		assertLenCap(t, got, 0, 0)
	})

	t.Run("nil source", func(t *testing.T) {
		var src []int
		got := copyAs[[]int](t, src)
		assertLenCap(t, got, 0, 0)
	})

	t.Run("backing arrays are independent", func(t *testing.T) {
		src := []int{1, 2, 3}
		dst := copyAs[[]int](t, src)

		dst[0] = 100
		if src[0] != 1 {
			t.Errorf("mutating dst changed src: %v", src)
		}
		src[1] = 200
		if dst[1] != 2 {
			t.Errorf("mutating src changed dst: %v", dst)
		}
	})

	t.Run("preserves length and capacity", func(t *testing.T) {
		src := make([]int, 2, 10)
		src[0], src[1] = 1, 2
		dst := copyAs[[]int](t, src)
		assertEqual(t, dst, []int{1, 2})
		assertLenCap(t, dst, 2, 10)
	})

	t.Run("sub-slice uses its own len and cap, not the parent's", func(t *testing.T) {
		base := []int{1, 2, 3, 4, 5}
		src := base[1:3] // len 2, cap 4
		dst := copyAs[[]int](t, src)
		assertEqual(t, dst, []int{2, 3})
		assertLenCap(t, dst, 2, 4)
	})

	t.Run("overwrites existing target without touching its old backing array", func(t *testing.T) {
		dst := []int{9, 9, 9, 9}
		alias := dst
		if err := Copy([]int{1, 2}, &dst); err != nil {
			t.Fatal(err)
		}
		assertEqual(t, dst, []int{1, 2})
		assertEqual(t, alias, []int{9, 9, 9, 9})
	})

	t.Run("pointer to slice as source", func(t *testing.T) {
		src := []int{1, 2, 3}
		assertEqual(t, copyAs[[]int](t, &src), []int{1, 2, 3})
	})

	t.Run("pointer to slice as target", func(t *testing.T) {
		var dst *[]int
		if err := Copy([]int{1, 2}, &dst); err != nil {
			t.Fatal(err)
		}
		if dst == nil {
			t.Fatal("dst is nil")
		}
		assertEqual(t, *dst, []int{1, 2})
	})

	t.Run("named slice type to plain slice and back", func(t *testing.T) {
		assertEqual(t, copyAs[[]int](t, ids{1, 2}), []int{1, 2})
		assertEqual(t, copyAs[ids](t, []int{1, 2}), ids{1, 2})
	})
}

func TestCopy_ArrayToSlice(t *testing.T) {
	t.Run("array value", func(t *testing.T) {
		got := copyAs[[]int](t, [3]int{1, 2, 3})
		assertEqual(t, got, []int{1, 2, 3})
		assertLenCap(t, got, 3, 3)
	})

	t.Run("pointer to array", func(t *testing.T) {
		src := [2]string{"a", "b"}
		assertEqual(t, copyAs[[]string](t, &src), []string{"a", "b"})
	})

	t.Run("zero length array", func(t *testing.T) {
		assertLenCap(t, copyAs[[]int](t, [0]int{}), 0, 0)
	})
}

func TestCopy_NestedSlices(t *testing.T) {
	t.Run("deep copies inner slices", func(t *testing.T) {
		src := [][]int{{1, 2}, {3}}
		dst := copyAs[[][]int](t, src)
		assertEqual(t, dst, src)

		dst[0][0] = 100
		if src[0][0] != 1 {
			t.Errorf("inner slice is shared: src = %v", src)
		}
	})

	t.Run("nil and empty inner slices", func(t *testing.T) {
		src := [][]int{nil, {}, {1}}
		dst := copyAs[[][]int](t, src)
		if len(dst) != 3 {
			t.Fatalf("len = %d, want 3", len(dst))
		}
		assertLenCap(t, dst[0], 0, 0)
		assertLenCap(t, dst[1], 0, 0)
		assertEqual(t, dst[2], []int{1})
	})

	t.Run("three levels deep", func(t *testing.T) {
		src := [][][]int{{{1}, {2, 3}}, {{4}}}
		dst := copyAs[[][][]int](t, src)
		assertEqual(t, dst, src)
		dst[0][1][0] = 100
		if src[0][1][0] != 2 {
			t.Error("deepest slice is shared")
		}
	})

	t.Run("array of slices", func(t *testing.T) {
		src := [2][]int{{1}, {2, 3}}
		assertEqual(t, copyAs[[][]int](t, src), [][]int{{1}, {2, 3}})
	})
}

func TestCopy_SliceIntoInterface(t *testing.T) {
	t.Run("[]int becomes []any", func(t *testing.T) {
		assertEqual(t, copyAs[any](t, []int{1, 2, 3}), any([]any{1, 2, 3}))
	})

	t.Run("[]string becomes []any", func(t *testing.T) {
		assertEqual(t, copyAs[any](t, []string{"a", "b"}), any([]any{"a", "b"}))
	})

	t.Run("array becomes []any", func(t *testing.T) {
		got := copyAs[any](t, [3]int{1, 2, 3})
		assertEqual(t, got, any([]any{1, 2, 3}))
		assertLenCap(t, got, 3, 3)
	})

	t.Run("empty slice becomes empty non-nil []any", func(t *testing.T) {
		got, ok := copyAs[any](t, []int{}).([]any)
		if !ok {
			t.Fatal("result is not []any")
		}
		assertLenCap(t, got, 0, 0)
	})

	t.Run("nil slice becomes empty []any", func(t *testing.T) {
		var src []int
		got, ok := copyAs[any](t, src).([]any)
		if !ok {
			t.Fatal("result is not []any")
		}
		assertLenCap(t, got, 0, 0)
	})

	t.Run("preserves length and capacity", func(t *testing.T) {
		src := make([]int, 2, 10)
		got := copyAs[any](t, src)
		assertLenCap(t, got, 2, 10)
	})

	t.Run("nested slices become nested []any", func(t *testing.T) {
		got := copyAs[any](t, [][]int{{1, 2}, {3}})
		assertEqual(t, got, any([]any{[]any{1, 2}, []any{3}}))
	})

	t.Run("pointer to slice into interface", func(t *testing.T) {
		src := []int{1, 2}
		assertEqual(t, copyAs[any](t, &src), any([]any{1, 2}))
	})

	t.Run("result does not alias source", func(t *testing.T) {
		src := []int{1, 2}
		got := copyAs[any](t, src).([]any)
		src[0] = 100
		assertEqual(t, got[0], any(1))
	})
}

func TestCopy_InterfaceSource(t *testing.T) {
	t.Run("[]any into []int", func(t *testing.T) {
		assertEqual(t, copyAs[[]int](t, []any{1, 2, 3}), []int{1, 2, 3})
	})

	t.Run("[]int into []any", func(t *testing.T) {
		assertEqual(t, copyAs[[]any](t, []int{1, 2, 3}), []any{1, 2, 3})
	})

	t.Run("[]any heterogeneous into any", func(t *testing.T) {
		src := []any{1, "a", 2.5, true}
		assertEqual(t, copyAs[any](t, src), any([]any{1, "a", 2.5, true}))
	})

	t.Run("[]any containing slices into any", func(t *testing.T) {
		src := []any{[]int{1, 2}, []string{"a"}}
		assertEqual(t, copyAs[any](t, src), any([]any{[]any{1, 2}, []any{"a"}}))
	})

	t.Run("[]any containing []any into [][]int", func(t *testing.T) {
		src := []any{[]any{1, 2}, []any{3}}
		assertEqual(t, copyAs[[][]int](t, src), [][]int{{1, 2}, {3}})
	})

	t.Run("[]any containing pointers is dereferenced", func(t *testing.T) {
		a, b := 1, 2
		assertEqual(t, copyAs[[]int](t, []any{&a, &b}), []int{1, 2})
	})

	t.Run("pointer to interface holding a slice", func(t *testing.T) {
		var src any = []int{1, 2}
		assertEqual(t, copyAs[[]int](t, &src), []int{1, 2})
	})

	t.Run("pointer to interface holding a scalar", func(t *testing.T) {
		var src any = 42
		assertEqual(t, copyAs[int](t, &src), 42)
	})

	t.Run("[]error from concrete error values", func(t *testing.T) {
		got := copyAs[[]error](t, []myErr{{"a"}, {"b"}})
		if len(got) != 2 || got[0].Error() != "a" || got[1].Error() != "b" {
			t.Errorf("unexpected result: %v", got)
		}
	})
}

func TestCopy_PointerElements(t *testing.T) {
	t.Run("[]*int deep copies pointees", func(t *testing.T) {
		a, b := 1, 2
		src := []*int{&a, &b}
		dst := copyAs[[]*int](t, src)

		if len(dst) != 2 || *dst[0] != 1 || *dst[1] != 2 {
			t.Fatalf("unexpected values: %v", dst)
		}
		if dst[0] == src[0] || dst[1] == src[1] {
			t.Error("pointers were copied by address, expected fresh allocations")
		}
		*dst[0] = 100
		if a != 1 {
			t.Errorf("writing through dst changed source pointee: a = %d", a)
		}
	})

	t.Run("[]*int into []int", func(t *testing.T) {
		a, b := 1, 2
		assertEqual(t, copyAs[[]int](t, []*int{&a, &b}), []int{1, 2})
	})

	t.Run("[]int into []*int", func(t *testing.T) {
		dst := copyAs[[]*int](t, []int{1, 2})
		if len(dst) != 2 || dst[0] == nil || dst[1] == nil || *dst[0] != 1 || *dst[1] != 2 {
			t.Errorf("unexpected result: %v", dst)
		}
	})

	t.Run("[]*int into any", func(t *testing.T) {
		a, b := 1, 2
		assertEqual(t, copyAs[any](t, []*int{&a, &b}), any([]any{1, 2}))
	})
}

func TestCopy_SliceOfMapsIsShallow(t *testing.T) {
	m := map[string]int{"a": 1}
	dst := copyAs[[]map[string]int](t, []map[string]int{m})

	dst[0]["a"] = 2
	if m["a"] != 2 {
		t.Error("expected map elements to be shared (shallow copy)")
	}
}

func TestCopy_SliceIntoNonSliceReturnsError(t *testing.T) {
	cases := []struct {
		name   string
		src    any
		target any
	}{
		{"slice into int", []int{1}, new(int)},
		{"slice into string", []int{1}, new(string)},
		{"slice into map", []int{1}, new(map[string]int)},
		{"slice into struct", []int{1}, new(struct{})},
		{"slice into *int", []int{1}, new(*int)},
		{"array into int", [2]int{1, 2}, new(int)},
		{"empty slice into int", []int{}, new(int)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tt := reflect.TypeOf(tc.target).Elem()
			for tt.Kind() == reflect.Pointer {
				tt = tt.Elem()
			}
			err := Copy(tc.src, tc.target)
			assertErr(t, err, fmt.Sprintf("expected either slice or array, got %s", tt))
		})
	}
}

func TestCopy_NestedErrorsPropagate(t *testing.T) {
	t.Run("[][]int into []int", func(t *testing.T) {
		var dst []int
		err := Copy([][]int{{1}}, &dst)
		assertErr(t, err, "expected either slice or array, got int")
	})

	t.Run("[]any containing slice into []int", func(t *testing.T) {
		var dst []int
		err := Copy([]any{[]int{1}}, &dst)
		assertErr(t, err, "expected either slice or array, got int")
	})

	t.Run("error in a later element", func(t *testing.T) {
		var dst [][]int
		err := Copy([][]int{{1}, {2}, {3}}, &dst)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var bad []int
		err = Copy([][]int{{1}, {2}}, &bad)
		assertErr(t, err, "expected either slice or array, got int")
	})

	t.Run("no elements visited means no error", func(t *testing.T) {
		var dst []int
		if err := Copy([][]int{}, &dst); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestCopy_PanicsOnUnsupportedInputs(t *testing.T) {
	t.Run("untyped nil target", func(t *testing.T) {
		assertPanics(t, func() { _ = Copy(1, nil) })
	})

	t.Run("typed nil pointer target", func(t *testing.T) {
		assertPanics(t, func() { _ = Copy(1, (*int)(nil)) })
	})

	t.Run("untyped nil source", func(t *testing.T) {
		assertPanics(t, func() {
			var dst int
			_ = Copy(nil, &dst)
		})
	})

	t.Run("typed nil pointer source", func(t *testing.T) {
		assertPanics(t, func() {
			var dst int
			_ = Copy((*int)(nil), &dst)
		})
	})

	t.Run("nil interface element in []any", func(t *testing.T) {
		assertPanics(t, func() {
			var dst []any
			_ = Copy([]any{nil}, &dst)
		})
	})

	t.Run("nil pointer element in slice", func(t *testing.T) {
		assertPanics(t, func() {
			var dst []*int
			_ = Copy([]*int{nil}, &dst)
		})
	})

	t.Run("array as target (Value.Grow only works on slices)", func(t *testing.T) {
		assertPanics(t, func() {
			var dst [3]int
			_ = Copy([]int{1, 2, 3}, &dst)
		})
		assertPanics(t, func() {
			var dst [3]int
			_ = Copy([3]int{1, 2, 3}, &dst)
		})
	})

	t.Run("scalar type mismatch int -> string", func(t *testing.T) {
		assertPanics(t, func() {
			var dst string
			_ = Copy(1, &dst)
		})
	})

	t.Run("scalar type mismatch int -> int64", func(t *testing.T) {
		assertPanics(t, func() {
			var dst int64
			_ = Copy(1, &dst)
		})
	})

	t.Run("named type -> underlying type", func(t *testing.T) {
		assertPanics(t, func() {
			var dst int
			_ = Copy(myInt(1), &dst)
		})
	})

	t.Run("element type mismatch []int -> []string", func(t *testing.T) {
		assertPanics(t, func() {
			var dst []string
			_ = Copy([]int{1}, &dst)
		})
	})

	t.Run("[]any with wrong dynamic type -> []int", func(t *testing.T) {
		assertPanics(t, func() {
			var dst []int
			_ = Copy([]any{"a"}, &dst)
		})
	})

	t.Run("slice into non-empty interface", func(t *testing.T) {
		assertPanics(t, func() {
			var dst fmt.Stringer
			_ = Copy([]int{1}, &dst)
		})
	})

	t.Run("concrete type that does not implement target interface", func(t *testing.T) {
		assertPanics(t, func() {
			var dst error
			_ = Copy(42, &dst)
		})
	})
}

func TestCopy_NonSliceSourceIntoSliceTargetPanics(t *testing.T) {
	cases := []struct {
		name string
		src  any
	}{
		{"int", 1},
		{"string", "s"},
		{"map", map[string]int{"a": 1}},
		{"struct", struct{}{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertPanics(t, func() {
				var dst []int
				_ = Copy(tc.src, &dst)
			})
		})
	}

	t.Run("scalar element inside a slice of slices", func(t *testing.T) {
		assertPanics(t, func() {
			var dst [][]int
			_ = Copy([]any{[]int{1}, 2}, &dst)
		})
	})
}

func TestCopy_TargetIsPartiallyWrittenOnError(t *testing.T) {
	t.Run("pointer target is allocated before the error", func(t *testing.T) {
		var p *int
		err := Copy([]int{1}, &p)
		assertErr(t, err, "expected either slice or array, got int")
		if p == nil {
			t.Error("expected p to have been allocated before the error")
		}
	})

	t.Run("slice target is resized before the error", func(t *testing.T) {
		var dst []int
		err := Copy([][]int{{1}, {2}}, &dst)
		assertErr(t, err, "expected either slice or array, got int")
		if len(dst) != 2 {
			t.Errorf("len(dst) = %d, want 2 (resized before element copy failed)", len(dst))
		}
	})
}

func TestCopy_Aliasing(t *testing.T) {
	t.Run("slice value as both source and target is safe", func(t *testing.T) {
		s := []int{1, 2, 3}
		if err := Copy(s, &s); err != nil {
			t.Fatal(err)
		}
		assertEqual(t, s, []int{1, 2, 3})
	})

	// KNOWN ISSUE: sv is addressable and refers to the same memory as tv, so
	// tv.SetZero() wipes the source before it is read.
	t.Run("pointer to the same slice as source and target wipes it", func(t *testing.T) {
		s := []int{1, 2, 3}
		if err := Copy(&s, &s); err != nil {
			t.Fatal(err)
		}
		assertLenCap(t, s, 0, 0)
	})
}

func TestCopy_MoreShapes(t *testing.T) {
	t.Run("overwrites existing value inside any target", func(t *testing.T) {
		var dst any = "old"
		if err := Copy([]int{1}, &dst); err != nil {
			t.Fatal(err)
		}
		assertEqual(t, dst, any([]any{1}))
	})

	t.Run("[][]int into []any", func(t *testing.T) {
		assertEqual(t, copyAs[[]any](t, [][]int{{1, 2}, {3}}), []any{[]any{1, 2}, []any{3}})
	})

	t.Run("array of arrays into [][]int", func(t *testing.T) {
		assertEqual(t, copyAs[[][]int](t, [2][2]int{{1, 2}, {3, 4}}), [][]int{{1, 2}, {3, 4}})
	})

	t.Run("struct with unexported fields", func(t *testing.T) {
		type S struct{ a, b int }
		assertEqual(t, copyAs[S](t, S{1, 2}), S{1, 2})
	})

	t.Run("channel is shared, not duplicated", func(t *testing.T) {
		src := make(chan int)
		if dst := copyAs[chan int](t, src); dst != src {
			t.Error("expected same channel")
		}
	})
}
