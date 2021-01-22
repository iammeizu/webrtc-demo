package middleware

import (
	uuid "github.com/satori/go.uuid"
	"strconv"
	"time"
)

func GenRequestId() string {
	ts := time.Now().Unix()
	u := uuid.NewV4()
	requestID := strconv.FormatInt(ts, 10) + "," + u.String()
	return requestID
}