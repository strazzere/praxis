package main

import (
	"fmt"
	"testing"
)

func TestRemove(t *testing.T) {
	a := []int{1}
	b := remove(a, 0)
	if len(b) != 0 {
		t.Fatalf("Expected len of %d but got len of %d", 0, len(b))
	}

	// Test out of bounds
	_ = remove(a, 100)
}

func TestMakeRange(t *testing.T) {
	test := makeRange(0, 2)

	if len(test) != 3 {
		t.Fatalf("Expected len of %d but got len of %d", 3, len(test))
	}

	if test[0] != 0 &&
		test[1] != 1 &&
		test[2] != 2 {
		t.Fatalf("Values for makeRange where not as expected : %+v", test)
	}
}
func TestStripPort(t *testing.T) {
	expectedPort := 12345
	built := fmt.Sprintf("%s:%d", "localhost", expectedPort)

	port, err := stripPort(built)
	if err != nil {
		t.Fatalf("Failed stripping port during test: %s", err)
	}

	if expectedPort != port {
		t.Fatalf("Expected %d but got %d", expectedPort, port)
	}

	_, err = stripPort("notport.com")
	if err == nil {
		t.Fatalf("Expected and error to be thrown!")
	}
}
