package types

import "seehuhn.de/go/xmp"

type Text struct {
	xmp.TextValue
}

type UnorderedArray[T any] struct{}

type OrderedArray[T any] struct{}

type ProperName struct{}

type Date struct{}
