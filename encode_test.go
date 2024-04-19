package xmp

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
	"testing"
)

func TestEncodeSimple(t *testing.T) {
	p := &Packet{
		Properties: map[xml.Name]Value{
			{Space: "http://ns.seehuhn.de/test/", Local: "testname"}: textValue{val: "testvalue"},
		},
	}

	body, err := p.Encode()
	if err != nil {
		t.Fatal(err)
	}

	bodyString := string(body)
	fmt.Println(bodyString)

	if !strings.Contains(bodyString, "<test:testname>testvalue</test:testname>") {
		t.Fatal("missing property")
	}
}

func TestEncodeURL(t *testing.T) {
	url, err := url.Parse("http://example.com/")
	if err != nil {
		t.Fatal(err)
	}
	p := &Packet{
		Properties: map[xml.Name]Value{
			{Space: "http://ns.seehuhn.de/test/", Local: "testname"}: uriValue{val: url},
		},
	}

	body, err := p.Encode()
	if err != nil {
		t.Fatal(err)
	}

	bodyString := string(body)
	fmt.Println(bodyString)

	if !strings.Contains(bodyString, "<test:testname rdf:resource=\"http://example.com/\"/>") {
		t.Fatal("missing property")
	}
}
