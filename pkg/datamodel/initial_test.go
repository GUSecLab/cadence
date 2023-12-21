package datamodel

import (
	"testing"
)

func TestFloatRand(t *testing.T) {

	// Test the Float32() method
	for i := 0; i < 100; i++ {
		f := Float32()
		if f < 0.0 {
			t.Errorf("PRNG.Float32() returned invalid value: %f", f)
		}
	}
}

func TestFloat64Rand(t *testing.T) {
	Seed(12345)
	// Test the Float64() method
	for i := 0; i < 100; i++ {
		f := Float64()
		if f < 0.0 {
			t.Errorf("PRNG.Float32() returned invalid value: %f", f)
		}
	}
}
func TestIntRand(t *testing.T) {
	Seed(12345)
	// Test the Intn() method with n=10
	for i := 0; i < 100; i++ {
		x := Int()
		if x < 0 {
			t.Errorf("PRNG.Int() returned invalid value: %d", x)
		}
	}
}
func TestIntnRand(t *testing.T) {
	Seed(12345)
	// Test the Intn() method with n=10
	for i := 0; i < 100; i++ {
		var n int64 = 10
		x := Intn(n)
		if x < 0 || x > n {
			t.Errorf("PRNG.Intn(%d) returned invalid value: %d", n, x)
		}
	}

}
