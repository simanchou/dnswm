package utils

import (
	uuid "github.com/satori/go.uuid"
)

func GenUUID() (uidString string, err error) {
	id, err := uuid.NewV4()
	if err != nil {
		return
	}

	uidString = id.String()
	return
}
