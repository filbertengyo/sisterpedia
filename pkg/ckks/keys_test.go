package ckks

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestP1Keygen(t *testing.T) {
	p1, err := GetProfile("p1")
	if err != nil {
		t.Errorf("unexpected error when retrieving profile p1")
		return
	}

	_, err = GenKeys(*p1)
	if err != nil {
		t.Errorf("error while generating keys: %v", err)
		return
	}
}

func TestP2Keygen(t *testing.T) {
	p2, err := GetProfile("p1")
	if err != nil {
		t.Errorf("unexpected error when retrieving profile p2")
		return
	}

	_, err = GenKeys(*p2)
	if err != nil {
		t.Errorf("error while generating keys: %v", err)
		return
	}
}

func TestPublicKeysetBinaryMarshalling(t *testing.T) {
	profile, err := GetProfile("p1")
	if err != nil {
		t.Errorf("unexpected error when retrieving profile")
		return
	}

	keyset, err := GenKeys(*profile)
	if err != nil {
		t.Errorf("error while generating keys: %v", err)
		return
	}

	publicKeyset := keyset.ExportPublic()
	payload, err := publicKeyset.MarshalBinary()
	if err != nil {
		t.Errorf("error while encoding public keyset: %v", err)
		return
	}

	var recastedKeyset PublicKeyset
	err = recastedKeyset.UnmarshalBinary(payload)
	if err != nil {
		t.Errorf("error while decoding public keyset: %v", err)
		return
	}

	recastedPayload, err := recastedKeyset.MarshalBinary()
	if err != nil {
		t.Errorf("error while reencoding public keyset: %v", err)
		return
	}

	if !bytes.Equal(payload, recastedPayload) {
		t.Errorf("original and reconstructed payloads do not match")
	}
}

func TestPublicKeysetJsonMarshalling(t *testing.T) {
	profile, err := GetProfile("p1")
	if err != nil {
		t.Errorf("unexpected error when retrieving profile")
		return
	}

	keyset, err := GenKeys(*profile)
	if err != nil {
		t.Errorf("error while generating keys: %v", err)
		return
	}

	publicKeyset := keyset.ExportPublic()
	payload, err := json.Marshal(publicKeyset)
	if err != nil {
		t.Errorf("error while encoding public keyset: %v", err)
		return
	}

	var recastedKeyset PublicKeyset
	err = json.Unmarshal(payload, &recastedKeyset)
	if err != nil {
		t.Errorf("error while decoding public keyset: %v", err)
		return
	}

	recastedPayload, err := json.Marshal(recastedKeyset)
	if err != nil {
		t.Errorf("error while reencoding public keyset: %v", err)
		return
	}

	if !bytes.Equal(payload, recastedPayload) {
		t.Errorf("original and reconstructed payloads do not match")
	}
}

func TestKeysetBinaryMarshalling(t *testing.T) {
	profile, err := GetProfile("p1")
	if err != nil {
		t.Errorf("unexpected error when retrieving profile")
		return
	}

	keyset, err := GenKeys(*profile)
	if err != nil {
		t.Errorf("error while generating keys: %v", err)
		return
	}

	payload, err := keyset.MarshalBinary()
	if err != nil {
		t.Errorf("error while encoding keyset: %v", err)
		return
	}

	var recastedKeyset Keyset
	err = recastedKeyset.UnmarshalBinary(payload)
	if err != nil {
		t.Errorf("error while decoding keyset: %v", err)
		return
	}

	recastedPayload, err := recastedKeyset.MarshalBinary()
	if err != nil {
		t.Errorf("error while reencoding keyset: %v", err)
		return
	}

	if !bytes.Equal(payload, recastedPayload) {
		t.Errorf("original and reconstructed payloads do not match")
	}
}

func TestKeysetJsonMarshalling(t *testing.T) {
	profile, err := GetProfile("p1")
	if err != nil {
		t.Errorf("unexpected error when retrieving profile")
		return
	}

	keyset, err := GenKeys(*profile)
	if err != nil {
		t.Errorf("error while generating keys: %v", err)
		return
	}

	payload, err := json.Marshal(keyset)
	if err != nil {
		t.Errorf("error while encoding keyset: %v", err)
		return
	}

	var recastedKeyset Keyset
	err = json.Unmarshal(payload, &recastedKeyset)
	if err != nil {
		t.Errorf("error while decoding keyset: %v", err)
		return
	}

	recastedPayload, err := json.Marshal(recastedKeyset)
	if err != nil {
		t.Errorf("error while reencoding keyset: %v", err)
		return
	}

	if !bytes.Equal(payload, recastedPayload) {
		t.Errorf("original and reconstructed payloads do not match")
	}
}
