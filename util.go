package main

import (
	"encoding/binary"
	"math"
	"os"
	"strconv"
)

// itob returns an 8-byte big endian representation of v.
func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

// converts hex number (0x34be667) to floating point number
func hexToF(h string) float64 {
	f, _ := strconv.ParseUint(h[2:], 16, 32)
	return math.Round(float64(f)) / math.Pow(10, 6)
}

// get env value with default/fallback if doesn't exist
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
