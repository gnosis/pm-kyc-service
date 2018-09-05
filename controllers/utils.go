package controllers

import (
	"crypto/hmac"
	"crypto/sha1"
)

func CheckHmac(message, messageMAC, key []byte) bool {
	mac := hmac.New(sha1.New, key)
	mac.Write(message)
	expectedMAC := mac.Sum(nil)

	return hmac.Equal(messageMAC, expectedMAC)
}
