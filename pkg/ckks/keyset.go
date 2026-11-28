package ckks

import (
	"bytes"
	"encoding/gob"
	"encoding/json"

	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

type PublicKeyset struct {
	profile Profile
	pk      PublicKey
	rlk     RelinearizationKey
}

type serializedPublicKeyset struct {
	Profile Profile            `json:"profile"`
	PK      PublicKey          `json:"pk"`
	RLK     RelinearizationKey `json:"rlk"`
}

func (pks PublicKeyset) Profile() Profile {
	return pks.profile
}

func (pks PublicKeyset) PK() PublicKey {
	return pks.pk
}

func (pks PublicKeyset) RLK() RelinearizationKey {
	return pks.rlk
}

func (pks PublicKeyset) MarshalBinary() ([]byte, error) {
	payload := serializedPublicKeyset{
		Profile: pks.Profile(),
		PK:      pks.PK(),
		RLK:     pks.RLK(),
	}

	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)

	err := encoder.Encode(payload)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (pks *PublicKeyset) UnmarshalBinary(data []byte) error {
	buffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buffer)

	var payload serializedPublicKeyset
	err := decoder.Decode(&payload)
	if err != nil {
		return err
	}

	pks.profile = payload.Profile
	pks.pk = payload.PK
	pks.rlk = payload.RLK

	return nil
}

func (pks PublicKeyset) MarshalJSON() ([]byte, error) {
	return json.Marshal(serializedPublicKeyset{
		Profile: pks.Profile(),
		PK:      pks.PK(),
		RLK:     pks.RLK(),
	})
}

func (pks *PublicKeyset) UnmarshalJSON(data []byte) error {
	var payload serializedPublicKeyset
	err := json.Unmarshal(data, &payload)
	if err != nil {
		return err
	}

	pks.profile = payload.Profile
	pks.pk = payload.PK
	pks.rlk = payload.RLK

	return nil
}

type Keyset struct {
	profile Profile
	sk      SecretKey
	pk      PublicKey
	rlk     RelinearizationKey
}

type serializedKeyset struct {
	Profile Profile            `json:"profile"`
	SK      SecretKey          `json:"sk"`
	PK      PublicKey          `json:"pk"`
	RLK     RelinearizationKey `json:"rlk"`
}

func (ks Keyset) Profile() Profile {
	return ks.profile
}

func (ks Keyset) SK() SecretKey {
	return ks.sk
}

func (ks Keyset) PK() PublicKey {
	return ks.pk
}

func (ks Keyset) RLK() RelinearizationKey {
	return ks.rlk
}

func (ks Keyset) MarshalBinary() ([]byte, error) {
	payload := serializedKeyset{
		Profile: ks.Profile(),
		SK:      ks.SK(),
		PK:      ks.PK(),
		RLK:     ks.RLK(),
	}

	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)

	err := encoder.Encode(payload)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (ks *Keyset) UnmarshalBinary(data []byte) error {
	buffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buffer)

	var payload serializedKeyset
	err := decoder.Decode(&payload)
	if err != nil {
		return err
	}

	ks.profile = payload.Profile
	ks.sk = payload.SK
	ks.pk = payload.PK
	ks.rlk = payload.RLK

	return nil
}

func (ks Keyset) MarshalJSON() ([]byte, error) {
	return json.Marshal(serializedKeyset{
		Profile: ks.Profile(),
		SK:      ks.SK(),
		PK:      ks.PK(),
		RLK:     ks.RLK(),
	})
}

func (ks *Keyset) UnmarshalJSON(data []byte) error {
	var payload serializedKeyset
	err := json.Unmarshal(data, &payload)
	if err != nil {
		return err
	}

	ks.profile = payload.Profile
	ks.sk = payload.SK
	ks.pk = payload.PK
	ks.rlk = payload.RLK

	return nil
}

func (ks Keyset) ExportPublic() PublicKeyset {
	return PublicKeyset{
		profile: ks.Profile(),
		pk:      ks.PK(),
		rlk:     ks.RLK(),
	}
}

func (k Keyset) ExportSecret() SecretKey {
	return k.SK()
}

func (profile Profile) GenKeys() (Keyset, error) {
	kgen := ckks.NewKeyGenerator(profile.params)
	sk := kgen.GenSecretKeyNew()
	pk := kgen.GenPublicKeyNew(sk)
	rlk := kgen.GenRelinearizationKeyNew(sk)

	return Keyset{
		profile: profile,
		sk:      SecretKey{profile: profile, value: *sk},
		pk:      PublicKey{profile: profile, value: *pk},
		rlk:     RelinearizationKey{profile: profile, value: *rlk},
	}, nil
}
