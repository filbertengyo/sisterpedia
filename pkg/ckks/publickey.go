package ckks

import (
	"bytes"
	"encoding/gob"
	"encoding/json"

	"github.com/tuneinsight/lattigo/v6/core/rlwe"
)

type PublicKey struct {
	profile Profile
	value   rlwe.PublicKey
}

type serializedPublicKey struct {
	Profile   Profile `json:"profile"`
	PublicKey []byte  `json:"publicKey"`
}

func (pk PublicKey) Profile() Profile {
	return pk.profile
}

func (pk PublicKey) MarshalBinary() ([]byte, error) {
	publicKey, err := pk.value.MarshalBinary()
	if err != nil {
		return nil, err
	}

	payload := serializedPublicKey{
		Profile:   pk.Profile(),
		PublicKey: publicKey,
	}

	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)

	err = encoder.Encode(payload)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (pk *PublicKey) UnmarshalBinary(data []byte) error {
	buffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buffer)

	var payload serializedPublicKey
	err := decoder.Decode(&payload)
	if err != nil {
		return err
	}

	pk.profile = payload.Profile
	err = pk.value.UnmarshalBinary(payload.PublicKey)
	if err != nil {
		return err
	}

	return nil
}

func (pk PublicKey) MarshalJSON() ([]byte, error) {
	publicKey, err := pk.value.MarshalBinary()
	if err != nil {
		return nil, err
	}

	payload := serializedPublicKey{
		Profile:   pk.Profile(),
		PublicKey: publicKey,
	}

	return json.Marshal(payload)
}

func (pk *PublicKey) UnmarshalJSON(data []byte) error {
	var payload serializedPublicKey
	err := json.Unmarshal(data, &payload)
	if err != nil {
		return err
	}

	pk.profile = payload.Profile
	err = pk.value.UnmarshalBinary(payload.PublicKey)
	if err != nil {
		return err
	}

	return nil
}
