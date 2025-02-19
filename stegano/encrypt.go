package stegano

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"io"
	"slices"
)

// encrypt 加密一串数据。
// 加密后密文的格式：
//
//	 0                   1                   2                   3
//	 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|                         Random Key                            |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|         Nonce         |            Plaintext SHA1             |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|                       Encryption Data                         |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
func encrypt(plain []byte) (*bytes.Buffer, error) {
	key, nonce := make([]byte, 32), make([]byte, 12)
	_, _ = rand.Read(key)   // 生成密钥
	_, _ = rand.Read(nonce) // 生成向量

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	sum := sha1.Sum(key)
	seal := gcm.Seal(nil, nonce, plain, nil)

	buf := bytes.NewBuffer(key)
	buf.Write(nonce)
	buf.Write(sum[:])
	buf.Write(seal)

	return buf, nil
}

func dec(msg []byte) ([]byte, error) {
	const keySize, sumSize = 32, 20
	const size = keySize + sumSize
	payload := make([]byte, size)
	if n := copy(payload, msg); n != size {
		return nil, io.ErrShortBuffer
	}
	key := payload[:keySize]
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	const nonceSize = 12
	sum := sha1.Sum(key)
	nonce := sum[:nonceSize]

	raw, err := gcm.Open(nil, nonce, msg[size:], nil)
	if err != nil {
		return nil, err
	}
	check := sha1.Sum(raw)
	if !slices.Equal(check[:], payload[keySize:]) {
		return nil, io.ErrNoProgress
	}

	return raw, nil
}
