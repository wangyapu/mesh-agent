package server

import (
	"bytes"
	"encoding/binary"
)

/**
 * 协议头是16字节的定长数据
 * 2字节magic字符串0xdabb,0-7高位，8-15低位
 * 1字节的消息标志位。16-20序列id,21 event,22 two way,23请求或响应标识
 * 1字节状态。当消息类型为响应时，设置响应状态。24-31位。
 * 8字节，消息ID,long类型，32-95位。
 * 4字节，消息长度，96-127位
 **/
const (
	// header length.
	HEADER_LENGTH = 16

	// magic header
	MAGIC      = uint16(0xdabb)
	MAGIC_HIGH = byte(0xda)
	MAGIC_LOW  = byte(0xbb)

	// message flag.
	FLAG_REQUEST = byte(0x80)
	FLAG_TWOWAY  = byte(0x40)
	FLAG_EVENT   = byte(0x20) // for heartbeat

	SERIALIZATION_MASK = 0x1f

	DUBBO_VERSION = "2.0.1"
)

func PackRequest(totalBuf []byte, req *AgentRequest) ([]byte, error) {
	//attachments := make(map[string]string)
	//attachments["path"] = req.Interf

	startIndex := len(totalBuf)

	dubboHeader := [HEADER_LENGTH]byte{MAGIC_HIGH, MAGIC_LOW, FLAG_REQUEST | FLAG_TWOWAY}

	// magic
	totalBuf = append(totalBuf, dubboHeader[:]...)
	totalBuf[3+startIndex] = 0

	// serialization id, two way flag, event, request/response flag
	totalBuf[2+startIndex] |= byte(FLAG_REQUEST | 6)

	// request id
	binary.BigEndian.PutUint64(totalBuf[4+startIndex:], uint64(req.RequestID))

	totalBuf[2+startIndex] |= byte(FLAG_REQUEST)

	contentBuf := totalBuf[startIndex+HEADER_LENGTH:]

	contentBuf = append(contentBuf, '"')
	contentBuf = append(contentBuf, DUBBO_VERSION...)
	contentBuf = append(contentBuf, '"', '\n')

	contentBuf = append(contentBuf, '"')
	contentBuf = append(contentBuf, req.Interf...)
	contentBuf = append(contentBuf, '"', '\n')

	contentBuf = append(contentBuf, "null\n"...)

	contentBuf = append(contentBuf, '"')
	contentBuf = append(contentBuf, req.Method...)
	contentBuf = append(contentBuf, '"', '\n')

	contentBuf = append(contentBuf, '"')
	contentBuf = append(contentBuf, req.ParamType.String()...)
	contentBuf = append(contentBuf, '"', '\n')

	contentBuf = append(contentBuf, '"')
	contentBuf = append(contentBuf, req.Param...)
	contentBuf = append(contentBuf, '"', '\n')

	contentBuf = oneKeyValueToJson(contentBuf, "path", req.Interf)
	contentBuf = append(contentBuf, '\n')

	contentBufLen := len(contentBuf)
	binary.BigEndian.PutUint32(totalBuf[12+startIndex:], uint32(contentBufLen))

	totalBuf = totalBuf[:contentBufLen+startIndex+HEADER_LENGTH]

	return totalBuf, nil
}

func oneKeyValueToJson(buf []byte, key, value string) []byte {
	buf = append(buf, '{', '"')
	buf = append(buf, key...)
	buf = append(buf, '"', ':', '"')
	buf = append(buf, value...)
	buf = append(buf, '"', '}')
	return buf
}

func mapToJson(m map[string]string) string {
	var buf bytes.Buffer
	buf.WriteString("{")

	index := 0
	last := len(m) - 1

	for k, v := range m {
		buf.WriteString("\"")
		buf.WriteString(k)
		buf.WriteString("\":\"")
		buf.WriteString(v)
		buf.WriteString("\"")

		if index < last {
			buf.WriteString(",")
		}
		index++
	}

	buf.WriteString("}")

	return buf.String()
}
