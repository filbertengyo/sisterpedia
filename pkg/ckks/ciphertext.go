package ckks

import (
	"bytes"
	"encoding/gob"
	"encoding/json"

	"github.com/tuneinsight/lattigo/v6/core/rlwe"
)

type Ciphertext struct {
	profile Profile
	value   rlwe.Ciphertext
}

type serializedCiphertext struct {
	Profile    Profile `json:"profile"`
	Ciphertext []byte  `json:"ciphertext"`
}

func (c Ciphertext) Profile() Profile {
	return c.profile
}

func (c Ciphertext) MarshalBinary() ([]byte, error) {
	ciphertext, err := c.value.MarshalBinary()
	if err != nil {
		return nil, err
	}

	payload := serializedCiphertext{
		Profile: c.Profile(),
		Ciphertext: ciphertext,
	}

	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)

	err = encoder.Encode(payload)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (c *Ciphertext) UnmarshalBinary(data []byte) error {
	buffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buffer)

	var payload serializedCiphertext
	err := decoder.Decode(&payload)
	if err != nil {
		return err
	}

	c.profile = payload.Profile
	err = c.value.UnmarshalBinary(payload.Ciphertext)
	if err != nil {
		return err
	}

	return nil
}

func (c Ciphertext) MarshalJSON() ([]byte, error) {
	ciphertext, err := c.value.MarshalBinary()
	if err != nil {
		return nil, err
	}

	payload := serializedCiphertext{
		Profile:    c.Profile(),
		Ciphertext: ciphertext,
	}

	return json.Marshal(payload)
}

func (c *Ciphertext) UnmarshalJSON(data []byte) error {
	var payload serializedCiphertext
	err := json.Unmarshal(data, &payload)
	if err != nil {
		return err
	}

	c.profile = payload.Profile
	err = c.value.UnmarshalBinary(payload.Ciphertext)
	if err != nil {
		return err
	}

	return nil
}
