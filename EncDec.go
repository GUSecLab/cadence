package logic

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math"
)

var NodesAmount int

func GetNodesAmount() int {
	return NodesAmount
}

// generate random key
func generateRandomKey(keySize int) ([]byte, error) {
	key := make([]byte, keySize)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

// generate random iv
func generateRandomIV() ([]byte, error) {
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}
	return iv, nil
}

// decrypted a message
func DecryptMessage(p string, key []byte, iv []byte) string {
	cipherText, _ := base64.StdEncoding.DecodeString(p)
	block, _ := aes.NewCipher(key)
	stream := cipher.NewCTR(block, iv)
	decryptedText := make([]byte, len(cipherText))
	stream.XORKeyStream(decryptedText, cipherText)
	return string(decryptedText)
}

// for the secpartner logic, return the decrypted message string
func DecryptMessageNoKey(p string) string {
	key, err := generateRandomKey(32)
	if err != nil {
		fmt.Println("Error generating random key:", err)
		return ""
	}

	// Generate a random IV
	iv, err := generateRandomIV()
	if err != nil {
		fmt.Println("Error generating random IV:", err)
		return ""
	}

	cipherText, _ := base64.StdEncoding.DecodeString(p)
	block, _ := aes.NewCipher(key)
	stream := cipher.NewCTR(block, iv)
	decryptedText := make([]byte, len(cipherText))
	stream.XORKeyStream(decryptedText, cipherText)
	return string(decryptedText)
}

// encrypt a message
func EncryptMessage(key []byte, iv []byte, plaintextstring string) (string, error) {
	// Generate a new AES cipher using the provided key
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	plaintext := []byte(plaintextstring) //convert the string to a byte[]
	// Pad the plaintext to a multiple of the block size
	paddedPlaintext := make([]byte, len(plaintext)+(aes.BlockSize-len(plaintext)%aes.BlockSize))
	copy(paddedPlaintext, plaintext)

	// Create a new CBC mode block cipher using the AES cipher and IV
	mode := cipher.NewCBCEncrypter(block, iv)

	// Encrypt the padded plaintext
	ciphertext := make([]byte, len(paddedPlaintext))
	mode.CryptBlocks(ciphertext, paddedPlaintext)

	// Append the IV to the ciphertext
	ciphertext = append(iv, ciphertext...)

	// Encode the ciphertext as a base64 string and return it
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func EncryptMessageNoKey(message string) (string, error) {
	// Generate a random 256-bit (32-byte) key
	key, err := generateRandomKey(32)
	if err != nil {
		return "", err
	}

	// Generate a random IV
	iv, err := generateRandomIV()
	if err != nil {
		return "", err
	}

	// Convert the message to a byte slice
	plaintext := []byte(message)

	// Generate a new AES cipher using the random key
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// Pad the plaintext to a multiple of the block size
	paddedPlaintext := make([]byte, len(plaintext)+(aes.BlockSize-len(plaintext)%aes.BlockSize))
	copy(paddedPlaintext, plaintext)

	// Create a new CBC mode block cipher using the AES cipher and random IV
	mode := cipher.NewCBCEncrypter(block, iv)

	// Encrypt the padded plaintext
	ciphertext := make([]byte, len(paddedPlaintext))
	mode.CryptBlocks(ciphertext, paddedPlaintext)

	// Concatenate the IV and ciphertext and encode the result as a base64 string
	encoded := base64.StdEncoding.EncodeToString(append(iv, ciphertext...))

	return encoded, nil
}

// a function that guesses if a string is encrypted with aes
func isAESEncrypted(s string) bool {
	// Decode the base64-encoded ciphertext
	ciphertext, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return false
	}

	// Check if the ciphertext is long enough to contain a salt and/or IV
	if len(ciphertext) < aes.BlockSize {
		return false
	}

	// Extract the salt and IV from the ciphertext
	iv := ciphertext[aes.BlockSize : aes.BlockSize*2]

	// Attempt to decrypt the ciphertext using a dummy key and the extracted salt and IV
	block, err := aes.NewCipher(make([]byte, aes.BlockSize))
	if err != nil {
		return false
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize*2:], ciphertext[aes.BlockSize*2:])

	// If the decryption succeeds, assume the ciphertext is encrypted with AES
	return true
}

// a function that checks the entropy of a string
// if it is too high, the string is likely to be encrypted
func ShannonEntropy(s string) float64 {
	freq := make(map[byte]int)
	for i := 0; i < len(s); i++ {
		freq[s[i]]++
	}
	entropy := 0.0
	for _, count := range freq {
		p := float64(count) / float64(len(s))
		entropy -= p * math.Log2(p)
	}
	return entropy
}
