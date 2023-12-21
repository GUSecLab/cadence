package datamodel

import (
	"crypto/rand"
	"encoding/binary"
	"math"

	"golang.org/x/crypto/chacha20"
)

var key [32]byte
var nonce [12]byte
var Chacha *chacha20.Cipher

func Seed(seed int64) {
	// Seed the key with the input seed
	binary.LittleEndian.PutUint64(key[0:], uint64(seed))

	// Generate a random 96-bit nonce
	if _, err := rand.Read(nonce[:]); err != nil {
		panic(err)
	}

}

func Float32() float32 {
	var buf [4]byte
	Chacha, err := chacha20.NewUnauthenticatedCipher(key[:], nonce[:])
	if err != nil {
		panic(err)
	}
	Chacha.XORKeyStream(buf[:], buf[:])
	nonce[8]++
	if nonce[8] == 0 {
		nonce[9]++
	}
	return float32(binary.LittleEndian.Uint32(buf[:])) / math.MaxUint32
}

func Float64() float64 {
	var buf [8]byte
	Chacha, err := chacha20.NewUnauthenticatedCipher(key[:], nonce[:])
	if err != nil {
		panic(err)
	}
	Chacha.XORKeyStream(buf[:], buf[:])
	nonce[8]++
	if nonce[8] == 0 {
		nonce[9]++
	}
	return float64(binary.LittleEndian.Uint64(buf[:])) / math.MaxUint64

}

func Int() int64 {
	var buf [8]byte
	Chacha, err := chacha20.NewUnauthenticatedCipher(key[:], nonce[:])
	if err != nil {
		panic(err)
	}
	Chacha.XORKeyStream(buf[:], buf[:])
	nonce[8]++
	if nonce[8] == 0 {
		nonce[9]++
	}
	uint64Value := binary.LittleEndian.Uint64(buf[:])
	return int64(uint64Value & (1<<63 - 1))
}

func Intn(m int64) int64 {

	if m <= 0 { //a case when no randomness is needed
		return 0
	}
	return Int() % m
}

func Perm(n int) []int {
	// Create a slice of integers from 0 to n-1
	indexes := make([]int, n)
	for i := range indexes {
		indexes[i] = i
	}

	// Shuffle the slice using chacha20 as the PRNG
	for i := n - 1; i > 0; i-- {
		// Generate a random index in [0, i] using chacha20
		j := Intn(int64(i + 1))

		// Swap the current element with the random element
		indexes[i], indexes[j] = indexes[j], indexes[i]
	}

	return indexes
}

// shuffles a slice
func Shuffle(slice []interface{}) {
	// Get the permutation of indices
	perm := Perm(len(slice))

	// Create a new slice to hold the shuffled objects
	shuffled := make([]interface{}, len(slice))

	// Shuffle the objects according to the permutation
	for i, j := range perm {
		shuffled[j] = slice[i]
	}

	// Copy the shuffled slice back to the original slice
	copy(slice, shuffled)
}
