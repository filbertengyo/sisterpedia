package ckks

import (
	"encoding/base64"
	"encoding/json"

	"github.com/tuneinsight/lattigo/v6/core/rlwe"
)

type Ciphertext struct {
	value rlwe.Ciphertext
}

func (c Ciphertext) MarshalBinary() ([]byte, error) {
	return c.value.MarshalBinary()
}

func (c *Ciphertext) UnmarshalBinary(data []byte) error {
	return c.value.UnmarshalBinary(data)
}

func (c Ciphertext) MarshalJSON() ([]byte, error) {
	bin, err := c.MarshalBinary()
	if err != nil {
		return nil, err
	}

	b64 := base64.StdEncoding.EncodeToString(bin)

	return json.Marshal(b64)
}

func (c *Ciphertext) UnmarshalJSON(data []byte) error {
	var b64 string
	err := json.Unmarshal(data, &b64)
	if err != nil {
		return err
	}

	bin, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return err
	}

	return c.UnmarshalBinary(bin)
}
