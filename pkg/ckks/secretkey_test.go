package ckks

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestSecretKeyBinaryMarshalling(t *testing.T) {
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

	key := keyset.SK()
	payload, err := key.MarshalBinary()
	if err != nil {
		t.Errorf("error while encoding key: %v", err)
		return
	}

	var recastedKey SecretKey
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

func TestSecretKeyJsonMarshalling(t *testing.T) {
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

	key := keyset.SK()
	payload, err := json.Marshal(key)
	if err != nil {
		t.Errorf("error while encoding key: %v", err)
		return
	}

	var recastedKey SecretKey
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
