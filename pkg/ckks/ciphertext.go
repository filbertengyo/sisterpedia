package ckks

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"

	"github.com/tuneinsight/lattigo/v6/core/rlwe"
)

type Ciphertext struct {
	Profile Profile
	value   rlwe.Ciphertext
}

// Magic Number: SISTER + CT + (version)
const ciphertextMagicNumber0 = "SISTERCT0"

func (c Ciphertext) MarshalBinary() ([]byte, error) {
	profileBin, err := c.Profile.MarshalBinary()
	if err != nil {
		return nil, err
	}
	profileSize := make([]byte, 4)
	binary.BigEndian.PutUint32(profileSize, uint32(len(profileBin)))

	ciphertextBin, err := c.value.MarshalBinary()
	if err != nil {
		return nil, err
	}
	ciphertextSize := make([]byte, 4)
	binary.BigEndian.PutUint32(ciphertextSize, uint32(len(ciphertextBin)))

	chunks := [][]byte{
		[]byte(ciphertextMagicNumber0),
		profileSize,
		profileBin,
		ciphertextSize,
		ciphertextBin,
	}

	return bytes.Join(chunks, nil), nil
}

func (c *Ciphertext) UnmarshalBinary(data []byte) error {
	var err error = nil
	offset := 0

	if bytes.HasPrefix(data, []byte(ciphertextMagicNumber0)) {
		offset = len(ciphertextMagicNumber0)

		if offset+4 > len(data) {
			return errors.New("unexpected eof")
		}
		profileSize := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4

		if offset+profileSize > len(data) {
			return errors.New("unexpected eof")
		}
		profileBin := data[offset : offset+profileSize]
		offset += profileSize

		var profile Profile
		err = profile.UnmarshalBinary(profileBin)
		if err != nil {
			return err
		}

		if offset+4 > len(data) {
			return errors.New("unexpected eof")
		}
		ciphertextSize := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4

		if offset+ciphertextSize > len(data) {
			return errors.New("unexpected eof")
		}
		ciphertextBin := data[offset : offset+ciphertextSize]
		offset += profileSize

		var ciphertext rlwe.Ciphertext
		err = ciphertext.UnmarshalBinary(ciphertextBin)
		if err != nil {
			return err
		}

		c.Profile = profile
		c.value = ciphertext

		return nil
	}

	return errors.New("unrecognized binary format")
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
