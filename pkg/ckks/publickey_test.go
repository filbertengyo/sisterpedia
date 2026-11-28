package ckks

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestPublicKeyBinaryMarshalling(t *testing.T) {
	profile, err := GetProfile("p1")
	if err != nil {
		t.Errorf("unexpected error when retrieving profile")
		return
	}

	keyset, err := profile.GenKeys()
	if err != nil {
		t.Errorf("error while generating keys: %v", err)
		return
	}

	key := keyset.PK()
	payload, err := key.MarshalBinary()
	if err != nil {
		t.Errorf("error while encoding key: %v", err)
		return
	}

	var recastedKey PublicKey
	err = recastedKey.UnmarshalBinary(payload)
	if err != nil {
		t.Errorf("error while decoding key: %v", err)
		return
	}

	recastedPayload, err := recastedKey.MarshalBinary()
	if err != nil {
		t.Errorf("error while reencoding key: %v", err)
		return
	}

	if !bytes.Equal(payload, recastedPayload) {
		t.Errorf("original and reconstructed payloads do not match")
	}
}

func TestPublicKeyJsonMarshalling(t *testing.T) {
	profile, err := GetProfile("p1")
	if err != nil {
		t.Errorf("unexpected error when retrieving profile")
		return
	}

	keyset, err := profile.GenKeys()
	if err != nil {
		t.Errorf("error while generating keys: %v", err)
		return
	}

	key := keyset.PK()
	payload, err := json.Marshal(key)
	if err != nil {
		t.Errorf("error while encoding key: %v", err)
		return
	}

	var recastedKey PublicKey
	err = json.Unmarshal(payload, &recastedKey)
	if err != nil {
		t.Errorf("error while decoding key: %v", err)
		return
	}

	recastedPayload, err := json.Marshal(recastedKey)
	if err != nil {
		t.Errorf("error while reencoding key: %v", err)
		return
	}

	if !bytes.Equal(payload, recastedPayload) {
		t.Errorf("original and reconstructed payloads do not match")
	}
}
