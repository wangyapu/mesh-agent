package server

import (
	"encoding/binary"
)

const (
	Response_OK                byte = 20
	Response_CLIENT_TIMEOUT    byte = 30
	Response_SERVER_TIMEOUT    byte = 31
	Response_BAD_REQUEST       byte = 40
	Response_BAD_RESPONSE      byte = 50
	Response_SERVICE_NOT_FOUND byte = 60
	Response_SERVICE_ERROR     byte = 70
	Response_SERVER_ERROR      byte = 80
	Response_CLIENT_ERROR      byte = 90

	RESPONSE_WITH_EXCEPTION int32 = 0
	RESPONSE_VALUE          int32 = 1
	RESPONSE_NULL_VALUE     int32 = 2
)

type Status uint8

func (s Status) OK() bool {
	return s == 20
}

func (s Status) String() string {
	switch s {
	case 20:
		return "OK"
	case 30:
		return "CLIENT_TIMEOUT"
	case 31:
		return "SERVER_TIMEOUT"
	case 40:
		return "BAD_REQUEST"
	case 50:
		return "BAD_RESPONSE"
	case 60:
		return "SERVICE_NOT_FOUND"
	case 70:
		return "SERVICE_ERROR"
	case 80:
		return "SERVER_ERROR"
	case 90:
		return "CLIENT_ERROR"
	case 100:
		return "SERVER_THREADPOOL_EXHAUSTED_ERROR"
	default:
		return "Uknown"
	}
}

type DubboResponse struct {
	ID     int64
	Status Status
	Data   []byte
}

func UnpackResponse(buf []byte) ([]byte, *DubboResponse, error) {
	readable := len(buf)

	if readable < HEADER_LENGTH {
		return buf, nil, ErrHeaderNotEnough
	}

	var err error

	header := buf[0:16]

	dataLen := header[12:16]
	dLen := int32(binary.BigEndian.Uint32(dataLen))
	tt := dLen + HEADER_LENGTH

	if int32(readable) < tt {
		return buf, nil, ErrHeaderNotEnough
	}

	res := &DubboResponse{}

	res.ID = int64(binary.BigEndian.Uint64(buf[4:12]))
	res.Status = Status(buf[3])
	res.Data = make([]byte, dLen-3)
	copy(res.Data, buf[HEADER_LENGTH+2:tt-1])

	return buf[tt:], res, err
}
