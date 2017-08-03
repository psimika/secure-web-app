package main

import (
	"log"

	"github.com/gorilla/securecookie"
)

const (
	defaultHashKeySize  = 64
	defaultBlockKeySize = 32
)

func validHashKey(key string) []byte {
	if key == "" {
		hashKey := securecookie.GenerateRandomKey(defaultHashKeySize)
		if hashKey == nil {
			log.Fatal("could not generate random hash key, exiting...")
		}
		log.Println("Note: Using auto generated hash key.")
		log.Println("Note: Values signed with this key will be invalidated on server restart. Use -hashkey='<strong random key>' instead.")
		return hashKey
	}
	hashKey := []byte(key)
	if len(hashKey) != 32 && len(hashKey) != 64 {
		log.Fatal("hash key should be 32 or 64 bytes, exiting...", len(hashKey))
	}
	return hashKey
}

func validBlockKey(key string) []byte {
	if key == "" {
		blockKey := securecookie.GenerateRandomKey(defaultBlockKeySize)
		if blockKey == nil {
			log.Fatal("could not generate random block key, exiting...")
		}
		log.Println("Note: Using auto generated block key.")
		log.Println("Note: Values encrypted with this key will be invalidated on server restart. Use -blockkey='<strong random key>' instead.")
		return blockKey
	}
	blockKey := []byte(key)
	if len(blockKey) != 32 {
		log.Fatal("block key should be 32 bytes, exiting...")
	}
	return blockKey
}
