package ckks

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"

	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

type Parameters struct {
	value ckks.Parameters
}

func (p Parameters) MarshalBinary() ([]byte, error) {
	return p.value.MarshalBinary()
}

func (p *Parameters) UnmarshalBinary(data []byte) error {
	return p.value.UnmarshalBinary(data)
}

func (p Parameters) MarshalJSON() ([]byte, error) {
	bin, err := p.MarshalBinary()
	if err != nil {
		return nil, err
	}

	b64 := base64.StdEncoding.EncodeToString(bin)

	return json.Marshal(b64)
}

func (p *Parameters) UnmarshalJSON(data []byte) error {
	var b64 string
	err := json.Unmarshal(data, &b64)
	if err != nil {
		return err
	}

	bin, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return err
	}

	return p.UnmarshalBinary(bin)
}

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
	Params Parameters         `json:"params"`
	PK     PublicKey          `json:"pk"`
	RLK    RelinearizationKey `json:"rlk"`
}

// Magic Number: SISTER + PKS + (version)
const publicKeysetMagicNumber0 = "SISTERPKS0"

func (pks PublicKeyset) MarshalBinary() ([]byte, error) {
	paramsBin, err := pks.Params.MarshalBinary()
	if err != nil {
		return nil, err
	}
	paramsSize := make([]byte, 4)
	binary.BigEndian.PutUint32(paramsSize, uint32(len(paramsBin)))

	pkBin, err := pks.PK.MarshalBinary()
	if err != nil {
		return nil, err
	}
	pkSize := make([]byte, 4)
	binary.BigEndian.PutUint32(pkSize, uint32(len(pkBin)))

	rlkBin, err := pks.RLK.MarshalBinary()
	if err != nil {
		return nil, err
	}
	rlkSize := make([]byte, 4)
	binary.BigEndian.PutUint32(rlkSize, uint32(len(rlkBin)))

	chunks := [][]byte{
		[]byte(publicKeysetMagicNumber0),
		paramsSize,
		paramsBin,
		pkSize,
		pkBin,
		rlkSize,
		rlkBin,
	}

	return bytes.Join(chunks, nil), nil
}

func (pks *PublicKeyset) UnmarshalBinary(data []byte) error {
	var err error = nil
	offset := 0

	if bytes.HasPrefix(data, []byte(publicKeysetMagicNumber0)) {
		offset = len(publicKeysetMagicNumber0)

		if offset+4 > len(data) {
			return errors.New("unexpected eof")
		}
		paramsSize := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4

		if offset+paramsSize > len(data) {
			return errors.New("unexpected eof")
		}
		paramsBin := data[offset : offset+paramsSize]
		offset += paramsSize

		var params Parameters
		err = params.UnmarshalBinary(paramsBin)
		if err != nil {
			return err
		}

		if offset+4 > len(data) {
			return errors.New("unexpected eof")
		}
		pkSize := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4

		if offset+pkSize > len(data) {
			return errors.New("unexpected eof")
		}
		pkBin := data[offset : offset+pkSize]
		offset += pkSize

		var pk PublicKey
		err = pk.UnmarshalBinary(pkBin)
		if err != nil {
			return err
		}

		if offset+4 > len(data) {
			return errors.New("unexpected eof")
		}
		rlkSize := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4

		if offset+rlkSize > len(data) {
			return errors.New("unexpected eof")
		}
		rlkBin := data[offset : offset+rlkSize]
		offset += pkSize

		var rlk RelinearizationKey
		err = rlk.UnmarshalBinary(rlkBin)
		if err != nil {
			return err
		}

		pks.Params = params
		pks.PK = pk
		pks.RLK = rlk

		return nil
	}

	return errors.New("unrecognized binary format")
}

type Keyset struct {
	Params Parameters         `json:"params"`
	SK     SecretKey          `json:"sk"`
	PK     PublicKey          `json:"pk"`
	RLK    RelinearizationKey `json:"rlk"`
}

// Magic Number: SISTER + KS + (version)
const keysetMagicNumber0 = "SISTERKS0"

func (ks Keyset) MarshalBinary() ([]byte, error) {
	paramsBin, err := ks.Params.MarshalBinary()
	if err != nil {
		return nil, err
	}
	paramsSize := make([]byte, 4)
	binary.BigEndian.PutUint32(paramsSize, uint32(len(paramsBin)))

	skBin, err := ks.SK.MarshalBinary()
	if err != nil {
		return nil, err
	}
	skSize := make([]byte, 4)
	binary.BigEndian.PutUint32(skSize, uint32(len(skBin)))

	pkBin, err := ks.PK.MarshalBinary()
	if err != nil {
		return nil, err
	}
	pkSize := make([]byte, 4)
	binary.BigEndian.PutUint32(pkSize, uint32(len(pkBin)))

	rlkBin, err := ks.RLK.MarshalBinary()
	if err != nil {
		return nil, err
	}
	rlkSize := make([]byte, 4)
	binary.BigEndian.PutUint32(rlkSize, uint32(len(rlkBin)))

	chunks := [][]byte{
		[]byte(keysetMagicNumber0),
		paramsSize,
		paramsBin,
		skSize,
		skBin,
		pkSize,
		pkBin,
		rlkSize,
		rlkBin,
	}

	return bytes.Join(chunks, nil), nil
}

func (ks *Keyset) UnmarshalBinary(data []byte) error {
	var err error = nil
	offset := 0

	if bytes.HasPrefix(data, []byte(keysetMagicNumber0)) {
		offset = len(keysetMagicNumber0)

		if offset+4 > len(data) {
			return errors.New("unexpected eof")
		}
		paramsSize := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4

		if offset+paramsSize > len(data) {
			return errors.New("unexpected eof")
		}
		paramsBin := data[offset : offset+paramsSize]
		offset += paramsSize

		var params Parameters
		err = params.UnmarshalBinary(paramsBin)
		if err != nil {
			return err
		}

		if offset+4 > len(data) {
			return errors.New("unexpected eof")
		}
		skSize := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4

		if offset+skSize > len(data) {
			return errors.New("unexpected eof")
		}
		skBin := data[offset : offset+skSize]
		offset += skSize

		var sk SecretKey
		err = sk.UnmarshalBinary(skBin)
		if err != nil {
			return err
		}

		if offset+4 > len(data) {
			return errors.New("unexpected eof")
		}
		pkSize := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4

		if offset+pkSize > len(data) {
			return errors.New("unexpected eof")
		}
		pkBin := data[offset : offset+pkSize]
		offset += pkSize

		var pk PublicKey
		err = pk.UnmarshalBinary(pkBin)
		if err != nil {
			return err
		}

		if offset+4 > len(data) {
			return errors.New("unexpected eof")
		}
		rlkSize := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4

		if offset+rlkSize > len(data) {
			return errors.New("unexpected eof")
		}
		rlkBin := data[offset : offset+rlkSize]
		offset += pkSize

		var rlk RelinearizationKey
		err = rlk.UnmarshalBinary(rlkBin)
		if err != nil {
			return err
		}

		ks.Params = params
		ks.SK = sk
		ks.PK = pk
		ks.RLK = rlk

		return nil
	}

	return errors.New("unrecognized binary format")
}

func (ks Keyset) ExportPublic() PublicKeyset {
	return PublicKeyset{
		Params: ks.Params,
		PK:     ks.PK,
		RLK:    ks.RLK,
	}
}

func (k Keyset) ExportSecret() SecretKey {
	return k.SK
}

func GenKeys(profile Profile) (Keyset, error) {
	params, err := ckks.NewParametersFromLiteral(profile.asLattigoParams())
	if err != nil {
		return Keyset{}, err
	}

	kgen := ckks.NewKeyGenerator(params)
	sk := kgen.GenSecretKeyNew()
	pk := kgen.GenPublicKeyNew(sk)
	rlk := kgen.GenRelinearizationKeyNew(sk)

	return Keyset{
		Params: Parameters{value: params},
		SK:     SecretKey{value: *sk},
		PK:     PublicKey{value: *pk},
		RLK:    RelinearizationKey{value: *rlk},
	}, nil
}
