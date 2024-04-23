package types

import "seehuhn.de/go/xmp"

// Text represents a simple text value.
type Text struct {
	Value string
	xmp.Q
}

// GetXMP implements the [xmp.Type] interface.
func (t Text) GetXMP() xmp.Value {
	return xmp.TextValue{
		Value: t.Value,
		Q:     t.Q,
	}
}

// DecodeAnother implements the [xmp.Type] interface.
func (Text) DecodeAnother(val xmp.Value) (xmp.Type, error) {
	v, ok := val.(xmp.TextValue)
	if !ok {
		return nil, xmp.ErrInvalid
	}
	return Text{v.Value, v.Q}, nil
}

type UnorderedArray[E xmp.Type] struct {
	Val []E
	xmp.Q
}

func (u UnorderedArray[E]) GetXMP() xmp.Value {
	var vals []xmp.Value
	for _, v := range u.Val {
		vals = append(vals, v.GetXMP())
	}
	return xmp.ArrayValue{
		Value: vals,
		Type:  xmp.Unordered,
		Q:     u.Q,
	}
}

func (UnorderedArray[E]) DecodeAnother(val xmp.Value) (xmp.Type, error) {
	a, ok := val.(xmp.ArrayValue)
	if !ok || a.Type != xmp.Unordered {
		return nil, xmp.ErrInvalid
	}
	res := UnorderedArray[E]{Q: a.Q}
	res.Val = make([]E, len(a.Value))
	for i, val := range a.Value {
		w, err := res.Val[i].DecodeAnother(val)
		if err != nil {
			return nil, err
		}
		res.Val[i] = w.(E)
	}
	res.Q = a.Q
	return res, nil
}
