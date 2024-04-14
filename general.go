package xmp

import "encoding/xml"

type generalModel struct {
	nameSpace  string
	properties map[string]generalValue
}

func (g *generalModel) NameSpaces(m map[string]struct{}) map[string]struct{} {
	panic("not implemented")
}

func (g *generalModel) EncodeXMP(e *Encoder, prefix string) error {
	panic("not implemented")
}

type generalValue struct {
	Tokens []xml.Token
	Q
}

func (v generalValue) IsZero() bool {
	return false
}

func (v generalValue) NameSpaces(m map[string]struct{}) map[string]struct{} {
	m = v.Q.NameSpaces(m)
	for _, token := range v.Tokens {
		switch token := token.(type) {
		case xml.StartElement:
			m[token.Name.Space] = struct{}{}
		}
	}
	return m
}

func (v generalValue) EncodeXMP(e *Encoder) error {
	for _, token := range v.Tokens {
		err := e.EncodeToken(token)
		if err != nil {
			return err
		}
	}
	return nil
}

func (v generalValue) DecodeAnother(tokens []xml.Token) (Value, error) {
	return generalValue{Tokens: tokens}, nil
}
