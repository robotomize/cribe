package hashing

import (
	"crypto/md5" //nolint
	"crypto/sha1"
	"fmt"
	"hash"
)

type HashFunc func([]byte) ([]byte, error)

func HashSumFunc(hasher func() hash.Hash) HashFunc {
	return func(in []byte) ([]byte, error) {
		h := hasher()
		if _, err := h.Write(in); err != nil {
			return nil, fmt.Errorf("%T(hashfile.Hash) write: %w", h, err)
		}

		return h.Sum(nil), nil
	}
}

func MD5HashFunc() HashFunc {
	return HashSumFunc(func() hash.Hash {
		return md5.New()
	})
}

func SHA1HashFunc() HashFunc {
	return HashSumFunc(func() hash.Hash {
		return sha1.New()
	})
}
