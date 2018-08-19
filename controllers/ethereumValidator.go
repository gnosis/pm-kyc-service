package controllers

import (
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/astaxie/beego/logs"
	"github.com/ethereum/go-ethereum/crypto"
)

func isValidChecksum(s string) bool {
	// We assume input has valid domain, because it's checked by the router with a regex

	if len(s) != 40 {
		logs.Warning("Address: ", s, "     length: ", len(s))
		return false
	}

	var hashedAddress = hex.EncodeToString(crypto.Keccak256([]byte(strings.ToLower(s))))

	logs.Info("Checking if address ", s, " is checksum validating it's hash ", string(hashedAddress))

	for i := 0; i < 40; i++ {
		positionValue, _ := strconv.ParseInt(string(hashedAddress[i]), 16, 64) // value, base, bitSize
		if positionValue > 7 && string(s[i]) != strings.ToUpper(string(s[i])) {
			return false
		}
		if positionValue <= 7 && string(s[i]) != strings.ToLower(string(s[i])) {
			return false
		}
	}

	return true
}
