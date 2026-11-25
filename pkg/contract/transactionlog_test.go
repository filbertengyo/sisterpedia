package contract

import "testing"

type noisyCodec struct {
	noise float64
}

func (c noisyCodec) EncryptVector(values []float64) ([]float64, error) {
	return values, nil
}

func (c noisyCodec) DecryptVector(ciphertext []float64, size int) ([]float64, error) {
	plaintext := make([]float64, size)
	for i := range size {
		plaintext[i] = ciphertext[i] + c.noise
	}
	return plaintext, nil
}

func TestTransactionLogEncryptionRoundTrip(t *testing.T) {
	codec := noisyCodec{noise: WarnNoiseTolerance - 0.1}

	log := TransactionLog{
		Id:             "T001",
		ProductNumber:  "P001",
		Quantity:       67,
		DiscountNumber: "D001",
	}

	ciphertext, err := EncryptTransactionLog(codec, log)
	if err != nil {
		t.Errorf("error during encryption: %v", err)
		return
	}

	decryptedLog, err := DecryptTransactionLog(codec, ciphertext)
	if err != nil {
		t.Errorf("error during decryption: %v", err)
		return
	}

	if log != decryptedLog {
		t.Error("decrypted log is not equal")
		return
	}
}

func TestTransactionLogNoisyDecryption(t *testing.T) {
	codec := noisyCodec{noise: MaxNoiseTolerance - 0.1}

	log := TransactionLog{
		Id:             "T001",
		ProductNumber:  "P001",
		Quantity:       67,
		DiscountNumber: "D001",
	}

	ciphertext, err := EncryptTransactionLog(codec, log)
	if err != nil {
		t.Errorf("error during encryption: %v", err)
		return
	}

	noisyLog, err := DecryptTransactionLog(codec, ciphertext)
	if err != nil {
		t.Errorf("error during decryption: %v", err)
		return
	}

	if !noisyLog.IsNoisy {
		t.Error("noisy log is not reported")
		return
	}
}

func TestTransactionLogVeryNoisyDecryption(t *testing.T) {
	codec := noisyCodec{noise: MaxNoiseTolerance + 0.1}

	log := TransactionLog{
		Id:             "T001",
		ProductNumber:  "P001",
		Quantity:       67,
		DiscountNumber: "D001",
	}

	ciphertext, err := EncryptTransactionLog(codec, log)
	if err != nil {
		t.Errorf("error during encryption: %v", err)
		return
	}

	_, err = DecryptTransactionLog(codec, ciphertext)
	if err == nil {
		t.Error("very noisy ciphertext decryption is not aborted")
		return
	}
}
