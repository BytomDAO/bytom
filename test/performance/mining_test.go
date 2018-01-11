package performance

import (
	"testing"
)

func BenchmarkNewBlockTpl(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
	}
}
