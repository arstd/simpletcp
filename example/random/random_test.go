package random

import "testing"

func TestRandomString(t *testing.T) {
	t.Log(String(16))
}

func BenchmarkRandomString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		String(16)
	}
}
