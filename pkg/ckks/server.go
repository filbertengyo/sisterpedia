package ckks

import (
	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

type ServerCrypto struct {
	PublicKeyset PublicKeyset
	encoder      ckks.Encoder
	encryptor    rlwe.Encryptor
}

func (pks PublicKeyset) NewServerCrypto() (ServerCrypto, error) {
	encoder := *ckks.NewEncoder(pks.Profile.params)
	encryptor := *ckks.NewEncryptor(pks.Profile.params, &pks.PK.value)

	return ServerCrypto{
		PublicKeyset: pks,
		encoder:      encoder,
		encryptor:    encryptor,
	}, nil
}

func (sc ServerCrypto) EncryptScalar(value float64) (Ciphertext, error) {
	level := sc.PublicKeyset.Profile.params.MaxLevel()
	return sc.EncryptScalarAt(value, level)
}

func (sc ServerCrypto) EncryptVector(values []float64) (Ciphertext, error) {
	level := sc.PublicKeyset.Profile.params.MaxLevel()
	return sc.EncryptVectorAt(values, level)
}

func (sc ServerCrypto) EncryptScalarAt(value float64, level int) (Ciphertext, error) {
	return sc.EncryptVectorAt([]float64{value}, level)
}

func (sc ServerCrypto) EncryptVectorAt(values []float64, level int) (Ciphertext, error) {
	plaintext := ckks.NewPlaintext(sc.PublicKeyset.Profile.params, level)
	err := sc.encoder.Encode(values, plaintext)
	if err != nil {
		return Ciphertext{}, err
	}

	ciphertext, err := sc.encryptor.EncryptNew(plaintext)
	if err != nil {
		return Ciphertext{}, err
	}

	return Ciphertext{value: *ciphertext}, nil
}
