package main

import (
	"encoding/binary"
	"math"
	"os"
	"reflect"
	"strconv"
	"time"
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

func dfResample(data map[int64]float64, min time.Time, duration int64) map[int64]float64 {
	sampled := make(DF)
	mu := min.Unix() * 1000
	step := mu - mu%duration
	max := time.Now().Add(2*time.Hour).Unix() * 1000
	for ; step <= max; step = time.Unix(step/1000, 0).Add(1*time.Hour).Unix() * 1000 {
		a := sampled[step]
		a.sum = 0.0
		a.len = 0
		sampled[step] = a
	}
	ts := reflect.ValueOf(sampled).MapKeys()
	for i := 0; i < len(ts); i++ {
		t := ts[i].Int()
		var nt int64 = 0
		if i < len(ts)-1 {
			nt = ts[i+1].Int()
		} else {
			nt = t
		}
		for k, v := range data {
			if k >= t && k < nt {
				a := sampled[t]
				a.len = a.len + 1
				a.sum = a.sum + v
				sampled[t] = a
			}
		}
	}
	result := make(map[int64]float64)
	for k, v := range sampled {
		if v.len == 0 {
			result[k] = 0
		} else {
			result[k] = v.sum / float64(v.len)
		}
	}
	return result
}

func dfResampleInt(data map[int64]int, min time.Time, duration int64) map[int64]int {
	sampled := make(DF)
	mu := min.Unix() * 1000
	step := mu - mu%duration
	max := time.Now().Add(2*time.Hour).Unix() * 1000
	for ; step <= max; step = time.Unix(step/1000, 0).Add(1*time.Hour).Unix() * 1000 {
		a := sampled[step]
		a.sum = 0.0
		a.len = 0
		sampled[step] = a
	}
	ts := reflect.ValueOf(sampled).MapKeys()
	for i := 0; i < len(ts); i++ {
		t := ts[i].Int()
		var nt int64 = 0
		if i < len(ts)-1 {
			nt = ts[i+1].Int()
		} else {
			nt = t
		}
		for k, v := range data {
			if k >= t && k < nt {
				a := sampled[t]
				a.len = a.len + 1
				a.sum = a.sum + float64(v)
				sampled[t] = a
			}
		}
	}
	result := make(map[int64]int)
	for k, v := range sampled {
		if v.len == 0 {
			result[k] = 0
		} else {
			result[k] = int(math.Round(v.sum / float64(v.len)))
		}
	}
	return result
}
