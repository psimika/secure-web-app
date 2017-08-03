package https

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

//func NewCookie(name, value string, expires time.Duration) *http.Cookie {
//
//	oauthStateCookie := &http.Cookie{
//		Name:     name,
//		Value:    value,
//		Path:     "/",
//		Secure:   true,
//		HttpOnly: true,
//		Expires:  expires,
//	}
//	http.SetCookie(w, oauthStateCookie)
//}

func signCookieValue(value string, secret []byte) string {
	signedValue := signMessage([]byte(value), secret)
	return base64.URLEncoding.EncodeToString(signedValue)
}

func checkSignedCookieValue(value string, signedValue string, key []byte) (bool, error) {
	decodedSignedValue, err := base64.URLEncoding.DecodeString(signedValue)
	if err != nil {
		return false, fmt.Errorf("error base64 decoding cookie value: %v", err)
	}
	valid := checkMAC([]byte(value), decodedSignedValue, key)
	return valid, nil
}

func signMessage(message, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	return mac.Sum(nil)
}

// checkMAC reports whether messageMAC is a valid HMAC tag for message.
func checkMAC(message, messageMAC, key []byte) bool {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
	return hmac.Equal(messageMAC, expectedMAC)
}
