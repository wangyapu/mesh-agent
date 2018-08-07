package server

import (
	"errors"
)

const (
	Error     MessageType = 0x01
	Dubbo_Request               = 0x02
	Dubbo_Response              = 0x04
	Heartbeat             = 0x08
)

var (
	ErrHeaderNotEnough = errors.New("header buffer too short")
	ErrBodyNotEnough   = errors.New("body buffer too short")
	ErrIllegalPackage  = errors.New("illegal package!")
)

type MessageType int

//type Message struct {
//	ID          int64
//	Version     string
//	Type        MessageType
//	ServicePath string
//	Interface   string // Service
//	Method      string
//	Data        interface{}
//}
//
//type RpcInvocation struct {
//	Method         string
//	ParameterTypes string
//	Args           []byte
//	Attachments    map[string]string
//}
