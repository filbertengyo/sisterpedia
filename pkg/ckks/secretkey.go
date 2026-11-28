package ckks

import (
	"bytes"
	"encoding/gob"
	"encoding/json"

	"github.com/tuneinsight/lattigo/v6/core/rlwe"
)

type SecretKey struct {
	profile Profile
	value   rlwe.SecretKey
}

type serializedSecretKey struct {
	Profile   Profile `json:"profile"`
	SecretKey []byte  `json:"secretKey"`
}

func (sk SecretKey) Profile() Profile {
	return sk.profile
}

func (sk SecretKey) MarshalBinary() ([]byte, error) {
	secretKey, err := sk.value.MarshalBinary()
	if err != nil {
		return nil, err
	}

	payload := serializedSecretKey{
		Profile:   sk.Profile(),
		SecretKey: secretKey,
	}

	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)

	err = encoder.Encode(payload)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (sk *SecretKey) UnmarshalBinary(data []byte) error {
	buffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buffer)

	var payload serializedSecretKey
	err := decoder.Decode(&payload)
	if err != nil {
		return err
	}

	sk.profile = payload.Profile
	err = sk.value.UnmarshalBinary(payload.SecretKey)
	if err != nil {
		return err
	}

	return nil
}

func (sk SecretKey) MarshalJSON() ([]byte, error) {
	secretKey, err := sk.value.MarshalBinary()
	if err != nil {
		return nil, err
	}

	payload := serializedSecretKey{
		Profile:   sk.Profile(),
		SecretKey: secretKey,
	}

	return json.Marshal(payload)
}

func (sk *SecretKey) UnmarshalJSON(data []byte) error {
	var payload serializedSecretKey
	err := json.Unmarshal(data, &payload)
	if err != nil {
		return err
	}

	sk.profile = payload.Profile
	err = sk.value.UnmarshalBinary(payload.SecretKey)
	if err != nil {
		return err
	}

	return nil
}
