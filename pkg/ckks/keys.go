package ckks

import (
	"bytes"
	"encoding/gob"
	"encoding/json"

	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
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
		SK:      SecretKey{profile: profile, value: *sk},
		PK:      PublicKey{profile: profile, value: *pk},
		RLK:     RelinearizationKey{profile: profile, value: *rlk},
	}, nil
}
