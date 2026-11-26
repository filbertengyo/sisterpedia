package ckks

import (
	"bytes"
	"encoding/gob"
	"encoding/json"

	"github.com/tuneinsight/lattigo/v6/core/rlwe"
)

type Ciphertext struct {
	Profile Profile
	value   rlwe.Ciphertext
}

func (c Ciphertext) MarshalBinary() ([]byte, error) {
	type Alias Ciphertext

	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)

	err := encoder.Encode(Alias(c))
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (c *Ciphertext) UnmarshalBinary(data []byte) error {
	type Alias Ciphertext

	buffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buffer)

	err := decoder.Decode((*Alias)(c))
	if err != nil {
		return err
	}

	return nil
}

type ciphertextJson struct {
	Profile    string `json:"profile"`
	Ciphertext []byte `json:"ciphertext"`
}

func (c Ciphertext) MarshalJSON() ([]byte, error) {
	ciphertext, err := c.value.MarshalBinary()
	if err != nil {
		return nil, err
	}

	payload := ciphertextJson{
		Profile:    c.Profile.Name,
		Ciphertext: ciphertext,
	}

	return json.Marshal(payload)
}

func (c *Ciphertext) UnmarshalJSON(data []byte) error {
	var payload ciphertextJson
	err := json.Unmarshal(data, &payload)
	if err != nil {
		return err
	}

	c.Profile, err = GetProfile(payload.Profile)
	if err != nil {
		return err
	}

	err = c.value.UnmarshalBinary(payload.Ciphertext)
	if err != nil {
		return err
	}

	return nil
}
