package server

import (
	"testing"
	"time"
	"fmt"
	"encoding/json"
)

func TestEncodeRequest(t *testing.T) {
	buf := make([]byte, 0, (1024+256)*10)

	message := &AgentRequest{
		RequestID: 123,
		Interf:    "wyp",
		Method:    "abc",
		ParamType: ParamType_String,
	}

	t1 := time.Now()
	for i := 0; i < 1000000; i++ {
		buf, e := PackRequest(buf, message)

		if e != nil {
			// do something
		}

		UnpackResponse(buf)
	}

	elapsed := time.Since(t1)
	fmt.Println("encode decode elapsed: ", elapsed)
}

func TestMapToJson(t *testing.T) {
	a := make(map[string]string)
	a["abc"] = "123"
	a["abcd"] = "1234"

	t1 := time.Now()
	for i := 0; i < 1000000; i++ {
		json.Marshal(a)
	}
	elapsed := time.Since(t1)
	fmt.Println("json.Marshal elapsed: ", elapsed)

	t1 = time.Now()
	for i := 0; i < 1000000; i++ {
		mapToJson(a)
	}
	elapsed = time.Since(t1)
	fmt.Println("map to json elapsed: ", elapsed)
}

func TestLB(t *testing.T) {
	lb := &LoadBalancer{
		weight:        []uint32{1000, 1000, 1000},
		minWeight:     []uint32{1000, 1000, 1000},
		totalCount:    3,
		nowBase:       1000,
		lastCount:     3,
		lastIndex:     3,
		lastBaseCount: 3,
	}

	t1 := time.Now()
	for i := 0; i < 1000000; i++ {
		lb.Get()
	}
	elapsed := time.Since(t1)
	fmt.Println("lb get elapsed: ", elapsed)
}
