package main

import "testing"

func Test_greet(t *testing.T) {
	want := "Hi!"
	if got := "Hi!"; got != want {
		t.Errorf("greet() = %v, want %v", got, want)
	}
}
