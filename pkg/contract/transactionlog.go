package contract

import (
	"errors"
	"math"
	"slices"
)

type TransactionLog struct {
	Id             string
	ProductNumber  string
	Quantity       int
	DiscountNumber string
	IsNoisy        bool
}

const TransactionLogMagicNumber0 int = 515710 // SISTL0

func EncryptTransactionLog[CT any](encryptor encryptor[CT], log TransactionLog) (CT, error) {
	logId := encodeString(log.Id)
	productNumber := encodeString(log.ProductNumber)
	discountNumber := encodeString(log.DiscountNumber)

	vectorSize := len(logId) + len(productNumber) + len(discountNumber) + 6

	vector := slices.Concat(
		[]float64{float64(TransactionLogMagicNumber0)},
		[]float64{float64(vectorSize)},
		[]float64{float64(len(logId))},
		logId,
		[]float64{float64(len(productNumber))},
		productNumber,
		[]float64{float64(log.Quantity)},
		[]float64{float64(len(discountNumber))},
		discountNumber,
	)

	return encryptor.EncryptVector(vector)
}

const WarnNoiseTolerance float64 = 0.25
const MaxNoiseTolerance float64 = 0.4

func DecryptTransactionLog[CT any](decryptor decryptor[CT], ciphertext CT) (TransactionLog, error) {
	seedValues, err := decryptor.DecryptVector(ciphertext, 2)
	if err != nil {
		return TransactionLog{}, err
	}

	if int(math.Round(seedValues[0])) != TransactionLogMagicNumber0 {
		return TransactionLog{}, errors.New("transaction log magic number does not match")
	}

	vectorSize := int(math.Round(seedValues[1]))
	values, err := decryptor.DecryptVector(ciphertext, vectorSize)
	if err != nil {
		return TransactionLog{}, err
	}

	// Read Log ID

	offset := 2
	if offset+1 > len(values) {
		return TransactionLog{}, errors.New("unexpected end of vector")
	}

	logIdSizeValue := values[offset]
	logIdSize := int(math.Round(logIdSizeValue))

	offset += 1
	if offset+logIdSize > len(values) {
		return TransactionLog{}, errors.New("unexpected end of vector")
	}

	logIdValues := values[offset : offset+logIdSize]
	logId := decodeString(logIdValues)

	// Read Product Number

	offset += logIdSize
	if offset+1 > len(values) {
		return TransactionLog{}, errors.New("unexpected end of vector")
	}

	productNumberSizeValue := values[offset]
	productNumberSize := int(math.Round(productNumberSizeValue))

	offset += 1
	if offset+productNumberSize > len(values) {
		return TransactionLog{}, errors.New("unexpected end of vector")
	}

	productNumberValues := values[offset : offset+productNumberSize]
	productNumber := decodeString(productNumberValues)

	// Read Quantity

	offset += productNumberSize
	if offset+1 > len(values) {
		return TransactionLog{}, errors.New("unexpected end of vector")
	}

	quantityValue := values[offset]
	quantity := int(math.Round(quantityValue))

	// Read Discount Number

	offset += 1
	if offset+1 > len(values) {
		return TransactionLog{}, errors.New("unexpected end of vector")
	}

	discountNumberSizeValue := values[offset]
	discountNumberSize := int(math.Round(discountNumberSizeValue))

	offset += 1
	if offset+discountNumberSize > len(values) {
		return TransactionLog{}, errors.New("unexpected end of vector")
	}

	discountNumberValues := values[offset : offset+discountNumberSize]
	discountNumber := decodeString(discountNumberValues)

	noiseSources := slices.Concat(
		[]float64{
			logIdSizeValue,
			productNumberSizeValue,
			discountNumberSizeValue,
			quantityValue,
		},
		logIdValues,
		productNumberValues,
		discountNumberValues,
	)

	maxNoise := getMaxNoise(noiseSources)
	isNoisy := maxNoise > WarnNoiseTolerance

	if maxNoise > MaxNoiseTolerance {
		return TransactionLog{}, errors.New("ciphertext is too noisy")
	}

	return TransactionLog{
		Id:             logId,
		ProductNumber:  productNumber,
		Quantity:       quantity,
		DiscountNumber: discountNumber,
		IsNoisy:        isNoisy,
	}, nil
}
