package golike

import (
	"fmt"
	"testing"
)

var sink any

var sizes = []int{10, 1_000, 100_000}

func makeInts(n int) []int {
	s := make([]int, n)
	for i := range s {
		s[i] = i
	}
	return s
}

func makeStrings(n int) []string {
	s := make([]string, n)
	for i := range s {
		s[i] = fmt.Sprintf("item-%d", i)
	}
	return s
}

func makeMatrix(rows, cols int) [][]int {
	m := make([][]int, rows)
	for i := range m {
		m[i] = makeInts(cols)
	}
	return m
}

func mustCopy(b *testing.B, src, dst any) {
	if err := Copy(src, dst); err != nil {
		b.Fatal(err)
	}
}

func BenchmarkCopy_Int(b *testing.B) {
	var dst int
	b.ReportAllocs()
	for b.Loop() {
		mustCopy(b, 42, &dst)
	}
}

func BenchmarkCopy_String(b *testing.B) {
	var dst string
	b.ReportAllocs()
	for b.Loop() {
		mustCopy(b, "hello world", &dst)
	}
}

func BenchmarkCopy_Struct(b *testing.B) {
	type S struct {
		A int
		B string
		C float64
	}
	src := S{1, "x", 2.5}
	var dst S
	b.ReportAllocs()
	for b.Loop() {
		mustCopy(b, src, &dst)
	}
}

func BenchmarkCopy_IntoAny(b *testing.B) {
	var dst any
	b.ReportAllocs()
	for b.Loop() {
		mustCopy(b, 42, &dst)
	}
}

func BenchmarkCopy_PointerSource(b *testing.B) {
	x := 42
	var dst int
	b.ReportAllocs()
	for b.Loop() {
		mustCopy(b, &x, &dst)
	}
}

func BenchmarkCopy_PointerTarget(b *testing.B) {
	var dst *int
	b.ReportAllocs()
	for b.Loop() {
		mustCopy(b, 42, &dst)
	}
}

func BenchmarkCopy_DoublePointerTarget(b *testing.B) {
	var dst **int
	b.ReportAllocs()
	for b.Loop() {
		mustCopy(b, 42, &dst)
	}
}

func BenchmarkCopy_SliceInt(b *testing.B) {
	for _, n := range sizes {
		src := makeInts(n)

		b.Run(fmt.Sprintf("n=%d/Copy", n), func(b *testing.B) {
			var dst []int
			b.SetBytes(int64(n * 8))
			b.ReportAllocs()
			for range b.N {
				mustCopy(b, src, &dst)
			}
		})

		b.Run(fmt.Sprintf("n=%d/Baseline", n), func(b *testing.B) {
			b.SetBytes(int64(n * 8))
			b.ReportAllocs()
			for range b.N {
				dst := make([]int, len(src), cap(src))
				copy(dst, src)
				sink = dst
			}
		})
	}
}

func BenchmarkCopy_SliceString(b *testing.B) {
	for _, n := range sizes {
		src := makeStrings(n)

		b.Run(fmt.Sprintf("n=%d/Copy", n), func(b *testing.B) {
			var dst []string
			b.ReportAllocs()
			for range b.N {
				mustCopy(b, src, &dst)
			}
		})

		b.Run(fmt.Sprintf("n=%d/Baseline", n), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				dst := make([]string, len(src), cap(src))
				copy(dst, src)
				sink = dst
			}
		})
	}
}

func BenchmarkCopy_SliceStruct(b *testing.B) {
	type S struct {
		A int
		B string
	}
	for _, n := range sizes {
		src := make([]S, n)
		for i := range src {
			src[i] = S{i, "x"}
		}
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			var dst []S
			b.ReportAllocs()
			for range b.N {
				mustCopy(b, src, &dst)
			}
		})
	}
}

func BenchmarkCopy_ArrayToSlice(b *testing.B) {
	var src [1000]int
	for i := range src {
		src[i] = i
	}
	var dst []int
	b.ReportAllocs()
	for b.Loop() {
		mustCopy(b, src, &dst)
	}
}

func BenchmarkCopy_SliceIntoAny(b *testing.B) {
	for _, n := range sizes {
		src := makeInts(n)

		b.Run(fmt.Sprintf("n=%d/Copy", n), func(b *testing.B) {
			var dst any
			b.ReportAllocs()
			for range b.N {
				mustCopy(b, src, &dst)
			}
		})

		b.Run(fmt.Sprintf("n=%d/Baseline", n), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				dst := make([]any, len(src), cap(src))
				for i, v := range src {
					dst[i] = v
				}
				sink = dst
			}
		})
	}
}

func BenchmarkCopy_AnySliceToTyped(b *testing.B) {
	for _, n := range sizes {
		src := make([]any, n)
		for i := range src {
			src[i] = i
		}

		b.Run(fmt.Sprintf("n=%d/Copy", n), func(b *testing.B) {
			var dst []int
			b.ReportAllocs()
			for range b.N {
				mustCopy(b, src, &dst)
			}
		})

		b.Run(fmt.Sprintf("n=%d/Baseline", n), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				dst := make([]int, len(src), cap(src))
				for i, v := range src {
					dst[i] = v.(int)
				}
				sink = dst
			}
		})
	}
}

func BenchmarkCopy_HeterogeneousAnySlice(b *testing.B) {
	src := make([]any, 1000)
	for i := range src {
		switch i % 3 {
		case 0:
			src[i] = i
		case 1:
			src[i] = "s"
		default:
			src[i] = 1.5
		}
	}
	var dst any
	b.ReportAllocs()
	for b.Loop() {
		mustCopy(b, src, &dst)
	}
}

func BenchmarkCopy_NestedSlices(b *testing.B) {
	for _, dim := range []int{10, 100, 500} {
		src := makeMatrix(dim, dim)

		b.Run(fmt.Sprintf("%dx%d/Copy", dim, dim), func(b *testing.B) {
			var dst [][]int
			b.ReportAllocs()
			for range b.N {
				mustCopy(b, src, &dst)
			}
		})

		b.Run(fmt.Sprintf("%dx%d/Baseline", dim, dim), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				dst := make([][]int, len(src), cap(src))
				for i, row := range src {
					dst[i] = make([]int, len(row), cap(row))
					copy(dst[i], row)
				}
				sink = dst
			}
		})
	}
}

func BenchmarkCopy_NestedSlicesIntoAny(b *testing.B) {
	src := makeMatrix(100, 100)
	var dst any
	b.ReportAllocs()
	for b.Loop() {
		mustCopy(b, src, &dst)
	}
}

func BenchmarkCopy_PointerElements(b *testing.B) {
	for _, n := range []int{10, 1_000} {
		src := make([]*int, n)
		for i := range src {
			v := i
			src[i] = &v
		}

		b.Run(fmt.Sprintf("n=%d/Copy", n), func(b *testing.B) {
			var dst []*int
			b.ReportAllocs()
			for range b.N {
				mustCopy(b, src, &dst)
			}
		})

		b.Run(fmt.Sprintf("n=%d/Baseline", n), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				dst := make([]*int, len(src), cap(src))
				for i, p := range src {
					v := *p
					dst[i] = &v
				}
				sink = dst
			}
		})
	}
}

func BenchmarkCopy_Parallel(b *testing.B) {
	src := makeInts(1_000)
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var dst []int
		for pb.Next() {
			if err := Copy(src, &dst); err != nil {
				b.Error(err)
				return
			}
		}
	})
}
