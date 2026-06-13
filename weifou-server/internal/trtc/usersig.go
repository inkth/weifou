package trtc

import (
	"bytes"
	"compress/zlib"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// GenUserSig 生成 TRTC UserSig（TLSSigAPIv2 算法）。
func GenUserSig(sdkAppID int, secretKey, userID string, expire int) (string, error) {
	currTime := time.Now().Unix()
	sig := hmacSHA256(sdkAppID, secretKey, userID, currTime, int64(expire), "")

	obj := map[string]interface{}{
		"TLS.ver":        "2.0",
		"TLS.identifier": userID,
		"TLS.sdkappid":   sdkAppID,
		"TLS.expire":     expire,
		"TLS.time":       currTime,
		"TLS.sig":        sig,
	}
	raw, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}

	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	if _, err := w.Write(raw); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	return base64URLEncode(b.Bytes()), nil
}

func hmacSHA256(sdkAppID int, key, identifier string, currTime, expire int64, base64UserBuf string) string {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("TLS.identifier:%s\n", identifier))
	content.WriteString(fmt.Sprintf("TLS.sdkappid:%d\n", sdkAppID))
	content.WriteString(fmt.Sprintf("TLS.time:%d\n", currTime))
	content.WriteString(fmt.Sprintf("TLS.expire:%d\n", expire))
	if base64UserBuf != "" {
		content.WriteString(fmt.Sprintf("TLS.userbuf:%s\n", base64UserBuf))
	}
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(content.String()))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// 腾讯云特定的 base64 变体
func base64URLEncode(data []byte) string {
	s := base64.StdEncoding.EncodeToString(data)
	s = strings.ReplaceAll(s, "+", "*")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "=", "_")
	return s
}
