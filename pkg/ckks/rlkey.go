package ckks

import (
	"bytes"
	"encoding/gob"
	"encoding/json"

	"github.com/tuneinsight/lattigo/v6/core/rlwe"
)

type RelinearizationKey struct {
	profile Profile
	value   rlwe.RelinearizationKey
}

type serializedRelinearizationKey struct {
	Profile            Profile
	RelinearizationKey []byte
}

func (rlk RelinearizationKey) Profile() Profile {
	return rlk.profile
}

func (rlk RelinearizationKey) MarshalBinary() ([]byte, error) {
	relinearizationKey, err := rlk.value.MarshalBinary()
	if err != nil {
		return nil, err
	}

	payload := serializedRelinearizationKey{
		Profile:            rlk.Profile(),
		RelinearizationKey: relinearizationKey,
	}

	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)

	err = encoder.Encode(payload)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (rlk *RelinearizationKey) UnmarshalBinary(data []byte) error {
	buffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buffer)

	var payload serializedRelinearizationKey
	err := decoder.Decode(&payload)
	if err != nil {
		return err
	}

	rlk.profile = payload.Profile
	err = rlk.value.UnmarshalBinary(payload.RelinearizationKey)
	if err != nil {
		return err
	}

	return nil
}

func (rlk RelinearizationKey) MarshalJSON() ([]byte, error) {
	relinearizationKey, err := rlk.value.MarshalBinary()
	if err != nil {
		return nil, err
	}

	payload := serializedRelinearizationKey{
		Profile:            rlk.Profile(),
		RelinearizationKey: relinearizationKey,
	}

	return json.Marshal(payload)
}

func (rlk *RelinearizationKey) UnmarshalJSON(data []byte) error {
	var payload serializedRelinearizationKey
	err := json.Unmarshal(data, &payload)
	if err != nil {
		return err
	}

	rlk.profile = payload.Profile
	err = rlk.value.UnmarshalBinary(payload.RelinearizationKey)
	if err != nil {
		return err
	}

	return nil
}
