package golike

import (
	"fmt"
	"reflect"
)

type Converter func(src reflect.Value, target reflect.Value) error

type options struct {
	converter Converter
}

type Option func(*options)

func WithConverter(fn Converter) Option {
	return func(o *options) {
		o.converter = fn
	}
}

func Copy(src any, target any, opts ...Option) error {
	var fn Converter
	if opts != nil {
		options := &options{}
		for _, opt := range opts {
			opt(options)
		}
		fn = options.converter
	}
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Pointer {
		return fmt.Errorf("expected pointer, got %s", value.Type())
	}
	return cp(reflect.ValueOf(src), value.Elem(), fn)
}

func cp(sv reflect.Value, tv reflect.Value, fn Converter) error {
	for tv.Kind() == reflect.Pointer {
		tv.Set(reflect.New(tv.Type().Elem()))
		tv = tv.Elem()
	}

	for sv.Kind() == reflect.Pointer {
		sv = sv.Elem()
	}

	switch sv.Kind() {
	case reflect.Slice, reflect.Array:
		{
			switch tv.Kind() {
			case reflect.Interface:
				{
					c := reflect.MakeSlice(reflect.SliceOf(tv.Type()), sv.Len(), sv.Cap())
					for i := range sv.Len() {
						if err := cp(sv.Index(i), c.Index(i), fn); err != nil {
							return err
						}
					}
					tv.Set(c)
					return nil
				}
			case reflect.Slice, reflect.Array:
				{
					tv.SetZero()
					tv.Grow(sv.Cap())
					tv.SetLen(sv.Len())
					tv.SetCap(sv.Cap())
					for i := range sv.Len() {
						if err := cp(sv.Index(i), tv.Index(i), fn); err != nil {
							return err
						}
					}
					return nil
				}
			default:
				{
					return fmt.Errorf("expected either slice or array, got %s", tv.Type())
				}
			}
		}
	case reflect.Interface:
		{
			return cp(sv.Elem(), tv, fn)
		}
	default:
		{
			if fn != nil {
				return fn(sv, tv)
			}
			tv.Set(sv)
			return nil
		}
	}
}
