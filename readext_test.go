package xmp_test

import (
	"fmt"
	"testing"

	"seehuhn.de/go/xmp"
)

func TestReadLang(t *testing.T) {
	model, err := xmp.ReadFile("testdata/sample2.xml")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(model)
}
