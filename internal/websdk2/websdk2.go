package websdk2

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	DUOPrefix  = "TX"
	APPPrefix  = "APP"
	AUTHPrefix = "AUTH"

	DUOExpire = 300
	APPExpire = 3600

	IKeyLen = 20
	SKeyLen = 40
	AKeyLen = 40

	ErrUser = "ERR|The username specified is invalid."
	ErrIKey = "ERR|The Duo integration key specified is invalid."
	ErrSKey = "ERR|The Duo secret key specified is invalid."
	ErrAKey = "ERR|The application secret key specified must be at least 40 characters."
)

func signVals(key, vals, prefix string, expire int, currentTime time.Time) string {
	exp := currentTime.Unix() + int64(expire)
	val := fmt.Sprintf("%s|%d", vals, exp)
	b64 := base64.StdEncoding.EncodeToString([]byte(val))
	cookie := fmt.Sprintf("%s|%s", prefix, b64)

	sig := hmacSHA1(cookie, key)
	return fmt.Sprintf("%s|%s", cookie, sig)
}

func parseVals(key, val, prefix, ikey string, currentTime time.Time) string {
	ts := currentTime.Unix()

	parts := strings.Split(val, "|")
	if len(parts) != 3 {
		return ""
	}
	uPrefix := parts[0]
	uB64 := parts[1]
	uSig := parts[2]

	sig := hmacSHA1(uPrefix+"|"+uB64, key)
	if hmacSHA1(sig, key) != hmacSHA1(uSig, key) {
		return ""
	}

	if uPrefix != prefix {
		return ""
	}

	decoded, err := base64.StdEncoding.DecodeString(uB64)
	if err != nil {
		return ""
	}

	cookieParts := strings.Split(string(decoded), "|")
	if len(cookieParts) != 3 {
		return ""
	}
	user := cookieParts[0]
	uIkey := cookieParts[1]
	expStr := cookieParts[2]

	if uIkey != ikey {
		return ""
	}

	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil {
		return ""
	}

	if ts >= exp {
		return ""
	}

	return user
}

func hmacSHA1(data, key string) string {
	h := hmac.New(sha1.New, []byte(key))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// SignRequest generates a signed request for Duo authentication
func SignRequest(ikey, skey, akey, username string) string {
	if username == "" {
		return ErrUser
	}
	if strings.Contains(username, "|") {
		return ErrUser
	}
	if len(ikey) != IKeyLen {
		return ErrIKey
	}
	if len(skey) != SKeyLen {
		return ErrSKey
	}
	if len(akey) < AKeyLen {
		return ErrAKey
	}

	currentTime := time.Now()
	vals := fmt.Sprintf("%s|%s", username, ikey)

	duoSig := signVals(skey, vals, DUOPrefix, DUOExpire, currentTime)
	appSig := signVals(akey, vals, APPPrefix, APPExpire, currentTime)

	return fmt.Sprintf("%s:%s", duoSig, appSig)
}

// VerifyResponse verifies the Duo response signature
func VerifyResponse(ikey, skey, akey, sigResponse string) string {
	parts := strings.Split(sigResponse, ":")
	if len(parts) != 2 {
		return ""
	}
	authSig := parts[0]
	appSig := parts[1]

	currentTime := time.Now()
	authUser := parseVals(skey, authSig, AUTHPrefix, ikey, currentTime)
	appUser := parseVals(akey, appSig, APPPrefix, ikey, currentTime)

	if authUser != appUser || authUser == "" {
		return ""
	}

	return authUser
}

