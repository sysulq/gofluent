package ik

import (
	"strconv"
	"regexp"
	"errors"
	"math/rand"
	"time"
)

var capacityRegExp = regexp.MustCompile("^([0-9]+)([kKmMgGtTpPeE])?(i?[bB])?")

func ParseCapacityString(s string) (int64, error) {
	m := capacityRegExp.FindStringSubmatch(s)
	if m == nil {
		return -1, errors.New("Invalid format: " + s)
	}
	base := int64(1000)
	if len(m[3]) > 0 {
		if m[3][0] == 'i' {
			base = int64(1024)
		}
	}
	multiply := int64(1)
	if len(m[2]) > 0 {
		switch m[2][0] {
			case 'e', 'E':
				multiply *= base
				fallthrough
			case 'p', 'P':
				multiply *= base
				fallthrough
			case 't', 'T':
				multiply *= base
				fallthrough
			case 'g', 'G':
				multiply *= base
				fallthrough
			case 'm', 'M':
				multiply *= base
				fallthrough
			case 'k', 'K':
				multiply *= base
		}
	}
	i, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil || multiply * i < i {
		return -1, errors.New("Invalid format (out of range): " + s)
	}
	return multiply * i, nil
}

func NewRandSourceWithTimestampSeed() rand.Source {
	return rand.NewSource(time.Now().UnixNano())
}
