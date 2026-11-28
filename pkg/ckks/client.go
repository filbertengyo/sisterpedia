package ckks

import (
	"errors"

	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

type ClientCrypto struct {
	Keyset    Keyset
	decryptor rlwe.Decryptor
	ServerCrypto
}

func (ks Keyset) NewClientCrypto() (ClientCrypto, error) {
	encoder := *ckks.NewEncoder(ks.Profile().params)
	encryptor := *ckks.NewEncryptor(ks.Profile().params, &ks.pk.value)
	decryptor := *ckks.NewDecryptor(ks.Profile().params, &ks.sk.value)

	return ClientCrypto{
		Keyset:    ks,
		decryptor: decryptor,
		ServerCrypto: ServerCrypto{
			PublicKeyset: ks.ExportPublic(),
			encoder:      encoder,
			encryptor:    encryptor,
		},
	}, nil
}

func (cc ClientCrypto) DecryptScalar(ciphertext Ciphertext) (float64, error) {
	values, err := cc.DecryptVector(ciphertext, 1)
	if err != nil {
		return 0, err
	}

	if len(values) < 1 {
		return 0, errors.New("plaintext is shorter than expected")
	}

	return values[0], nil
}

func (cc ClientCrypto) DecryptVector(ciphertext Ciphertext, size int) ([]float64, error) {
	plaintext := cc.decryptor.DecryptNew(&ciphertext.value)

	values := make([]float64, size)
	err := cc.encoder.Decode(plaintext, values)
	if err != nil {
		return nil, err
	}

	return values, nil
}
