package ckks

import (
	"math"
	"testing"
)

func TestNewClientCrypto(t *testing.T) {
	profile, err := GetProfileByName("p1")
	if err != nil {
		t.Errorf("error when retrieving profile: %v", err)
		return
	}

	keyset, err := profile.GenKeys()
	if err != nil {
		t.Errorf("error generating keyset: %v", err)
		return
	}

	_, err = keyset.NewClientCrypto()
	if err != nil {
		t.Errorf("error obtaining client crypto: %v", err)
		return
	}
}

func TestDecryptScalar(t *testing.T) {
	profile, err := GetProfileByName("p1")
	if err != nil {
		t.Errorf("error when retrieving profile: %v", err)
		return
	}

	keyset, err := profile.GenKeys()
	if err != nil {
		t.Errorf("error generating keyset: %v", err)
		return
	}

	clientCrypto, err := keyset.NewClientCrypto()
	if err != nil {
		t.Errorf("error obtaining client crypto: %v", err)
		return
	}

	plaintext := float64(5761)

	ciphertext, err := clientCrypto.EncryptScalar(plaintext)
	if err != nil {
		t.Errorf("error during encryption: %v", err)
		return
	}

	decrypted, err := clientCrypto.DecryptScalar(ciphertext)
	if err != nil {
		t.Errorf("error during decryption: %v", err)
		return
	}

	dist := math.Abs(decrypted - plaintext)
	maxDist := 0.001
	if dist > maxDist {
		t.Errorf("distance between plaintext (%f) and decrypted (%f) is too large", plaintext, decrypted)
	}
}

func TestDecryptVector(t *testing.T) {
	profile, err := GetProfileByName("p1")
	if err != nil {
		t.Errorf("error when retrieving profile: %v", err)
		return
	}

	keyset, err := profile.GenKeys()
	if err != nil {
		t.Errorf("error generating keyset: %v", err)
		return
	}

	clientCrypto, err := keyset.NewClientCrypto()
	if err != nil {
		t.Errorf("error obtaining client crypto: %v", err)
		return
	}

	plaintext := []float64{5, 7, 6, 1}

	ciphertext, err := clientCrypto.EncryptVector(plaintext)
	if err != nil {
		t.Errorf("error during encryption: %v", err)
		return
	}

	decrypted, err := clientCrypto.DecryptVector(ciphertext, 4)
	if err != nil {
		t.Errorf("error during decryption: %v", err)
		return
	}

	maxDist := 0.001
	for i, _ := range plaintext {
		dist := math.Abs(decrypted[i] - plaintext[i])
		if dist > maxDist {
			t.Errorf("distance on element %d between plaintext (%f) and decrypted (%f) is too large", i, plaintext, decrypted)
		}
	}
}
