package ckks

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"

	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

type SecretKey struct {
	value rlwe.SecretKey
}

func (sk SecretKey) MarshalBinary() ([]byte, error) {
	return sk.value.MarshalBinary()
}

func (sk *SecretKey) UnmarshalBinary(data []byte) error {
	return sk.value.UnmarshalBinary(data)
}

func (sk SecretKey) MarshalJSON() ([]byte, error) {
	bin, err := sk.MarshalBinary()
	if err != nil {
		return nil, err
	}

	b64 := base64.StdEncoding.EncodeToString(bin)

	return json.Marshal(b64)
}

func (sk *SecretKey) UnmarshalJSON(data []byte) error {
	var b64 string
	err := json.Unmarshal(data, &b64)
	if err != nil {
		return err
	}

	bin, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return err
	}

	return sk.UnmarshalBinary(bin)
}

type PublicKey struct {
	value rlwe.PublicKey
}

func (pk PublicKey) MarshalBinary() ([]byte, error) {
	return pk.value.MarshalBinary()
}

func (pk *PublicKey) UnmarshalBinary(data []byte) error {
	return pk.value.UnmarshalBinary(data)
}

func (pk PublicKey) MarshalJSON() ([]byte, error) {
	bin, err := pk.MarshalBinary()
	if err != nil {
		return nil, err
	}

	b64 := base64.StdEncoding.EncodeToString(bin)

	return json.Marshal(b64)
}

func (pk *PublicKey) UnmarshalJSON(data []byte) error {
	var b64 string
	err := json.Unmarshal(data, &b64)
	if err != nil {
		return err
	}

	bin, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return err
	}

	return pk.UnmarshalBinary(bin)
}

type RelinearizationKey struct {
	value rlwe.RelinearizationKey
}

func (rlk RelinearizationKey) MarshalBinary() ([]byte, error) {
	return rlk.value.MarshalBinary()
}

func (rlk *RelinearizationKey) UnmarshalBinary(data []byte) error {
	return rlk.value.UnmarshalBinary(data)
}

func (rlk RelinearizationKey) MarshalJSON() ([]byte, error) {
	bin, err := rlk.MarshalBinary()
	if err != nil {
		return nil, err
	}

	b64 := base64.StdEncoding.EncodeToString(bin)

	return json.Marshal(b64)
}

func (rlk *RelinearizationKey) UnmarshalJSON(data []byte) error {
	var b64 string
	err := json.Unmarshal(data, &b64)
	if err != nil {
		return err
	}

	bin, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return err
	}

	return rlk.UnmarshalBinary(bin)
}

type PublicKeyset struct {
	Profile Profile            `json:"profile"`
	PK      PublicKey          `json:"pk"`
	RLK     RelinearizationKey `json:"rlk"`
}

func (pks PublicKeyset) MarshalBinary() ([]byte, error) {
	type Alias PublicKeyset

	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)

	err := encoder.Encode(Alias(pks))
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (pks *PublicKeyset) UnmarshalBinary(data []byte) error {
	type Alias PublicKeyset

	buffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buffer)

	err := decoder.Decode((*Alias)(pks))
	if err != nil {
		return err
	}

	return nil
}

type Keyset struct {
	Profile Profile            `json:"profile"`
	SK      SecretKey          `json:"sk"`
	PK      PublicKey          `json:"pk"`
	RLK     RelinearizationKey `json:"rlk"`
}

func (ks Keyset) MarshalBinary() ([]byte, error) {
	type Alias Keyset

	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)

	err := encoder.Encode(Alias(ks))
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (ks *Keyset) UnmarshalBinary(data []byte) error {
	type Alias Keyset

	buffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buffer)

	err := decoder.Decode((*Alias)(ks))
	if err != nil {
		return err
	}

	return nil
}

func (ks Keyset) ExportPublic() PublicKeyset {
	return PublicKeyset{
		Profile: ks.Profile,
		PK:      ks.PK,
		RLK:     ks.RLK,
	}
}

func (k Keyset) ExportSecret() SecretKey {
	return k.SK
}

func (profile Profile) GenKeys() (Keyset, error) {
	kgen := ckks.NewKeyGenerator(profile.params)
	sk := kgen.GenSecretKeyNew()
	pk := kgen.GenPublicKeyNew(sk)
	rlk := kgen.GenRelinearizationKeyNew(sk)

	return Keyset{
		Profile: profile,
		SK:      SecretKey{value: *sk},
		PK:      PublicKey{value: *pk},
		RLK:     RelinearizationKey{value: *rlk},
	}, nil
}
