/*
	Referenced site: https://blog.alexellis.io/golang-writing-unit-tests/
 */

package main

import (
	"testing"
)

func TestStripChars(t *testing.T) {
	tables := []struct {
		i string
		o string
	} {
		{"a b c d e", "abcde"}, 
		{"  a b    c  d    e", "abcde"},
	}
	
	for _, table := range tables {
		r := stripChars(table.i, " ")
	    if r != table.o {
	       t.Errorf("stripChars() was incorrect, got: %s, want: %s.", r, table.o)
		}
	}
}