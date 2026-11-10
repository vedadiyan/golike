# golike

[![Go Reference](https://pkg.go.dev/badge/github.com/vedadiyan/golike.svg)](https://pkg.go.dev/github.com/vedadiyan/golike)

A small, reflection-based helper for copying values in Go, with deep-copy semantics for slices, arrays and pointers and automatic conversion between typed slices and `[]any`.

```sh
go get github.com/vedadiyan/golike
```

Requires **Go 1.22+**.

## Usage

```go
func Copy(src any, target any) error
```

`target` must be a non-nil pointer. `src` may be a value or a pointer.

```go
package main

import (
	"fmt"

	"github.com/vedadiyan/golike"
)

func main() {
	src := [][]int{{1, 2}, {3}}

	var dst [][]int
	if err := golike.Copy(src, &dst); err != nil {
		panic(err)
	}

	dst[0][0] = 100
	fmt.Println(src[0][0]) // 1 — inner slices are independent
}
```

### More examples

```go
// Typed slice -> []any
var anyDst any
golike.Copy([]int{1, 2, 3}, &anyDst)
// anyDst == []any{1, 2, 3}

// Nested slices become nested []any
golike.Copy([][]int{{1, 2}, {3}}, &anyDst)
// anyDst == []any{[]any{1, 2}, []any{3}}

// []any -> typed slice
var ints []int
golike.Copy([]any{1, 2, 3}, &ints)

// Array -> slice
golike.Copy([3]int{1, 2, 3}, &ints)

// Pointers are dereferenced on the source side and allocated on the target side
x := 5
var p *int
golike.Copy(&x, &p) // p points to a new int, not to x

// []*int is deep-copied: pointees are cloned
a, b := 1, 2
var ptrs []*int
golike.Copy([]*int{&a, &b}, &ptrs)

// Pointer and non-pointer element types are interchangeable
golike.Copy([]*int{&a, &b}, &ints) // []int{1, 2}
golike.Copy([]int{1, 2}, &ptrs)    // []*int
```

## Semantics

| Source | Target | Result |
| --- | --- | --- |
| scalar, struct, map, chan, func | same type | assigned directly (**shallow**) |
| scalar or concrete value | `any` / interface it implements | stored in the interface |
| `[]T` / `[N]T` | `[]U` | new slice; each element copied recursively; **length and capacity preserved** |
| `[]T` / `[N]T` | `any` | `[]any`; elements copied recursively (nested slices become nested `[]any`) |
| `[]any` | `[]T` | each element's dynamic value is copied into `T` |
| `*T`, `**T`, … | anything | dereferenced before copying |
| anything | `*T`, `**T`, … | pointers are freshly allocated; existing pointees are never written through |
| slice or array | non-slice type | `expected either slice or array, got <type>` |
| any non-pointer target | | `expected pointer, got <type>` |

Source types are matched by *kind*, so named slice types work: `type IDs []int` copies to and from `[]int`.

## Limitations

These are the current behaviors, and they are covered by the test suite.

**Shallow outside slices.** Maps, channels, and slice fields inside structs are shared with the source, not cloned.

**Panics instead of errors** for:

- a nil source (untyped, or a nil pointer), or a nil element inside `[]any` / `[]*T`
- a nil target (untyped or typed nil pointer)
- element or scalar type mismatches (`int` → `string`, `int` → `int64`, `myInt` → `int`, `[]int` → `[]string`)
- a non-slice source going into a slice target
- a slice going into a non-empty interface such as `fmt.Stringer`
- **arrays as the target** (`*[3]int`); only slices are supported as targets

**Partial writes on error.** Target pointers are allocated and slice targets resized before elements are copied, so a returned error can leave the target partially written.

**Aliasing.** `Copy(s, &s)` is safe, but `Copy(&s, &s)` empties `s`, because the source and target refer to the same memory.

**Cyclic pointers** (e.g. `type P *P`) are not detected and will loop forever.

## Testing

```sh
go test ./...
```
