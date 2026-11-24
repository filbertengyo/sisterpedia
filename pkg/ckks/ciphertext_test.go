package ckks

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestCiphertextBinaryMarshal(t *testing.T) {
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

	ciphertext, err := clientCrypto.EncryptScalar(5761)
	if err != nil {
		t.Errorf("error obtaining ciphertext: %v", err)
		return
	}

	payload, err := ciphertext.MarshalBinary()
	if err != nil {
		t.Errorf("error encoding ciphertext: %v", err)
		return
	}

	var reformedCiphertext Ciphertext
	err = reformedCiphertext.UnmarshalBinary(payload)
	if err != nil {
		t.Errorf("error decoding ciphertext: %v", err)
		return
	}

	reformedPayload, err := reformedCiphertext.MarshalBinary()
	if err != nil {
		t.Errorf("error reencoding ciphertext: %v", err)
		return
	}

	if !bytes.Equal(payload, reformedPayload) {
		t.Error("reencoded ciphertext does not match original")
		return
	}
}

func TestCiphertextJsonMarshal(t *testing.T) {
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

	ciphertext, err := clientCrypto.EncryptScalar(5761)
	if err != nil {
		t.Errorf("error obtaining ciphertext: %v", err)
		return
	}

	payload, err := json.Marshal(ciphertext)
	if err != nil {
		t.Errorf("error encoding ciphertext: %v", err)
		return
	}

	var reformedCiphertext Ciphertext
	err = json.Unmarshal(payload, &reformedCiphertext)
	if err != nil {
		t.Errorf("error decoding ciphertext: %v", err)
		return
	}

	reformedPayload, err := json.Marshal(reformedCiphertext)
	if err != nil {
		t.Errorf("error reencoding ciphertext: %v", err)
		return
	}

	if !bytes.Equal(payload, reformedPayload) {
		t.Error("reencoded ciphertext does not match original")
		return
	}
}
