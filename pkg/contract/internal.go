package contract

import (
	"bytes"
	"encoding/binary"
	"math"
	"slices"
)

type encryptor[CT any] interface {
	EncryptVector(values []float64) (CT, error)
}

type decryptor[CT any] interface {
	DecryptVector(ciphertext CT, size int) ([]float64, error)
}

const packingSize = 6 // Gives at least 1/32 precision
const int64Size = 8

func encodeString(string string) []float64 {
	stringBytes := []byte(string)

	size := (len(string) + packingSize - 1) / packingSize
	values := make([]float64, size)

	i := 0
	for chunk := range slices.Chunk(stringBytes, packingSize) {
		chunkSize := len(chunk)
		paddingSize := int64Size - chunkSize
		padding := make([]byte, paddingSize, int64Size)

		data := append(chunk, padding...)
		intValue := binary.LittleEndian.Uint64(data)
		values[i] = float64(intValue)

		i++
	}

	return values
}

func decodeString(values []float64) string {
	capacity := len(values) * packingSize
	stringBytes := make([]byte, 0, capacity)

	for _, v := range values {
		intValue := uint64(math.Round(v))

		data := make([]byte, int64Size)
		binary.LittleEndian.PutUint64(data, intValue)

		paddingSize := int64Size - packingSize
		chunk := data[:len(data)-paddingSize]
		stringBytes = append(stringBytes, chunk...)
	}

	stringBytes = bytes.TrimRight(stringBytes, "\x00")
	return string(stringBytes)
}

func getMaxNoise(values []float64) float64 {
	maxNoise := 0.0
	for _, v := range values {
		noise := math.Abs(v - math.Round(v))
		maxNoise = math.Max(noise, maxNoise)
	}
	return maxNoise
}
