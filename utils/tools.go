package utils

import (
	"crypto/md5"
	"fmt"
	uuid "github.com/satori/go.uuid"
	"strings"
)

func GenUUID() (uidString string, err error) {
	id, err := uuid.NewV4()
	if err != nil {
		return
	}

	uidString = id.String()
	return
}

func MD5ID(str string) string {
	_str := strings.Join(strings.Fields(str), "")
	h := md5.Sum([]byte(strings.ToUpper(_str)))
	return fmt.Sprintf("%x", h)
}
