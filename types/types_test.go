package types

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/text/language"
	"seehuhn.de/go/xmp"
)

func TestText(t *testing.T) {
	p := xmp.NewPacket()

	A := Text{
		Value: "hello world",
		Q:     xmp.Q{xmp.Language(language.English)},
	}
	p.Set("http://ns.seehuhn.de/test/#", "prop", A)

	var B Text
	err := xmp.Get(&B, p, "http://ns.seehuhn.de/test/#", "prop")
	if err != nil {
		t.Fatalf("p.Get: %v", err)
	}

	if d := cmp.Diff(A, B); d != "" {
		t.Errorf("A and B are different (-want +got):\n%s", d)
	}
}

func TestUnorderedArray(t *testing.T) {
	p := xmp.NewPacket()

	A := UnorderedArray[Text]{
		Val: []Text{
			{Value: "Hello", Q: xmp.Q{xmp.Language(language.English)}},
			{Value: "Hallo", Q: xmp.Q{xmp.Language(language.German)}},
			{Value: "Bonjour", Q: xmp.Q{xmp.Language(language.French)}},
		},
	}
	p.Set("http://ns.seehuhn.de/test/#", "prop", A)

	var B UnorderedArray[Text]
	err := xmp.Get(&B, p, "http://ns.seehuhn.de/test/#", "prop")
	if err != nil {
		t.Fatalf("p.Get: %v", err)
	}

	if d := cmp.Diff(A, B); d != "" {
		t.Errorf("A and B are different (-want +got):\n%s", d)
	}
}
