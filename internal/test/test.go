package test

import "testing"

func Equal[K comparable](t *testing.T, want, have K, msg string) {
	t.Helper()

	if want != have {
		t.Fatalf("\n%s\nWant: %v\nHave: %v\n", msg, want, have)
	}
}

func HandleError(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}
