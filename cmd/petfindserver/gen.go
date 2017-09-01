// +build ignore

package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

const size = 64

func generateRandomKey(length int) []byte {
	k := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		return nil
	}
	return k
}

func main() {
	id := generateRandomKey(size)
	fmt.Println("len id", len(id))
	token := base64.URLEncoding.EncodeToString(id)
	fmt.Println("token:", token)
	fmt.Println("len token", len(token))

	//message := []byte("hello")
	//messageMAC := []byte("hello")
	secret := []byte("mysecret")
	m := sign(id, secret)
	fmt.Println("signed id", base64.URLEncoding.EncodeToString(m))
	fmt.Println("len signed id", len(base64.URLEncoding.EncodeToString(m)))
	//b := CheckMAC(message, m, key)
	//fmt.Println(b)
}

func sign(message, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	return mac.Sum(nil)
}

func CheckMAC(message, messageMAC, key []byte) bool {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
	return hmac.Equal(messageMAC, expectedMAC)
}
