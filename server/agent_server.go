package server

import (
	"encoding/binary"
	"errors"
	"fmt"
	"mesh-agent/config"
	"mesh-agent/logger"
)

const (
	agentPhase_Header    = 0
	agentPhase_Interface = 1
	agentPhase_Method    = 2
	agentPhase_ParamType = 3
	agentPhase_Param     = 4

	agentPacketMagic = 0x1F254798
)

var agentPacketParseError = errors.New("agentPacketParseError")

type ParamType int

const (
	ParamType_None   = iota
	ParamType_Int8
	ParamType_Int16
	ParamType_Int32
	ParamType_Uint8
	ParamType_Uint16
	ParamType_Uint32
	ParamType_String
	ParamType_Obj
	ParamType_Result
)

func (t ParamType) String() string {
	switch t {
	case ParamType_Int8:
	case ParamType_Int16:
	case ParamType_Int32:
	case ParamType_Uint8:
	case ParamType_Uint16:
	case ParamType_Uint32:
	case ParamType_String:
		return "Ljava/lang/String;"
	case ParamType_Obj:
		// TODO
	}
	return "int"
}

type AgentContext struct {
	ra  *RemoteAgent
	is  InputStream
	req *AgentRequest
}

type AgentRequest struct {
	conn     Conn
	paramLen int
	phase    int

	RequestID  uint64
	Result     uint32
	RemoteAddr string
	Interf     string
	Method     string
	ParamType  ParamType
	Param      []byte

	//profileRemoteAgentGetTime       time.Time
	//profileRemoteAgentSendDubboTime time.Time
	//profileRemoteAgentGetDubboTime  time.Time
	//profileRemoteAgentSendAgentTime time.Time
}

// 主动连接的Server需要特殊处理close 时间
// 这边需要处理的是agent 主动连接其他agent的连接
// 也就是consumer agent -> producer agent
func CreateAgentEvent(loops int, workerQueues []chan *AgentRequest, processorsNum uint64) *Events {
	events := &Events{}
	events.NumLoops = loops

	events.Serving = func(srv Server) (action Action) {
		logger.Info("agent server started (loops: %d)", srv.NumLoops)
		return
	}

	events.Opened = func(c Conn) (out []byte, opts Options, action Action) {
		if c.GetConnType() != config.ConnTypeAgent {
			return GlobalLocalDubboAgent.events.Opened(c)
		}
		lastCtx := c.Context()
		if lastCtx == nil {
			c.SetContext(&AgentContext{})
		}

		opts.ReuseInputBuffer = true

		logger.Info("agent opened: laddr: %v: raddr: %v", c.LocalAddr(), c.RemoteAddr())
		return
	}

	events.Closed = func(c Conn, err error) (action Action) {
		if c.GetConnType() != config.ConnTypeAgent {
			return GlobalLocalDubboAgent.events.Closed(c, err)
		}
		logger.Info("agent closed: %s: %s", c.LocalAddr(), c.RemoteAddr())
		return
	}

	events.Data = func(c Conn, in []byte) (out []byte, action Action) {
		if c.GetConnType() != config.ConnTypeAgent {
			return GlobalLocalDubboAgent.events.Data(c, in)
		}

		if in == nil {
			return
		}
		agentContext := c.Context().(*AgentContext)

		data := agentContext.is.Begin(in)

		// process the pipeline
		for {
			if len(data) > 0 {
				if agentContext.req == nil {
					agentContext.req = &AgentRequest{}
					agentContext.req.conn = c
					// handle the request
					//agentContext.req.RemoteAddr = c.RemoteAddr().String()
				}
			} else {
				break
			}

			leftover, err, ready := parseAgentReq(data, agentContext.req)

			if err != nil {
				// bad thing happened
				action = Close
				break
			} else if !ready {
				// request not ready, yet
				data = leftover
				break
			}

			//AppendRequest(out, httpContext.req)
			//agentContext.req.profileRemoteAgentGetTime = time.Now()
			//counterRecvConsumer.Incr(1)
			//counterRecvProvider.Incr(1)

			index := agentContext.req.RequestID % processorsNum
			workerQueues[index] <- agentContext.req
			agentContext.req = nil
			data = leftover
		}
		agentContext.is.End(data)
		return
	}
	return events
}

// provider agent server, 提供接口让其他人连接，只维护简单connection
func ServeListenAgent(loops int, port int, workerQueues []chan *AgentRequest) error {
	events := CreateAgentEvent(loops, workerQueues, uint64(len(workerQueues)))
	// We at least want the single http address.
	addrs := []string{fmt.Sprintf("tcp://0.0.0.0:%d?reuseport=true", port)}
	// Start serving!
	ser, err := ServeAndReturn(config.ConnTypeAgent, *events, addrs...)
	GlobalLocalDubboAgent.server = ser
	return err
}

func PackAgentRequest(totalBuf []byte,
	result uint32,
	reqID uint64,
	interf string,
	method string,
	paramType ParamType,
	param []byte) ([]byte, error) {
	rawLen := len(totalBuf)
	newLen := rawLen + 26 + len(interf) + len(method) + len(param)
	out := totalBuf[rawLen:newLen]
	buf := out
	binary.LittleEndian.PutUint32(buf, uint32(agentPacketMagic))
	buf = buf[4:]
	binary.LittleEndian.PutUint32(buf, result)
	buf = buf[4:]
	binary.LittleEndian.PutUint64(buf, reqID)
	buf = buf[8:]
	binary.LittleEndian.PutUint16(buf, uint16(len(interf)))
	buf = buf[2:]
	copy(buf[:], interf)
	buf = buf[len(interf):]
	binary.LittleEndian.PutUint16(buf, uint16(len(method)))
	buf = buf[2:]
	copy(buf, method)
	buf = buf[len(method):]
	binary.LittleEndian.PutUint16(buf, uint16(paramType))
	buf = buf[2:]

	binary.LittleEndian.PutUint32(buf, uint32(len(param)))
	buf = buf[4:]
	copy(buf, param)
	//fmt.Printf("send buffer %v \n", out)
	return totalBuf[0:newLen], nil
}

func parseAgentReq(data []byte, req *AgentRequest) (body []byte, err error, ready bool) {
	body = data
	for {
		switch req.phase {
		case agentPhase_Header:
			if len(body) < 16 {
				return body, err, false
			}

			magic := binary.LittleEndian.Uint32(body)
			if magic != agentPacketMagic {
				return body, agentPacketParseError, false
			}
			req.Result = binary.LittleEndian.Uint32(body[4:])
			req.RequestID = binary.LittleEndian.Uint64(body[8:])
			req.phase = agentPhase_Interface
			body = body[16:]
		case agentPhase_Interface:
			if len(body) < 2 {
				return body, err, false
			}
			interfaceLen := binary.LittleEndian.Uint16(body)
			if len(body)-2 < int(interfaceLen) {
				return body, err, false
			}
			req.Interf = string(body[2:(2 + interfaceLen)])
			body = body[2+interfaceLen:]
			req.phase = agentPhase_Method
		case agentPhase_Method:
			if len(body) < 2 {
				return body, err, false
			}
			methodLen := binary.LittleEndian.Uint16(body)
			if len(body)-2 < int(methodLen) {
				return body, err, false
			}
			req.Method = string(body[2:(2 + methodLen)])
			body = body[2+methodLen:]
			req.phase = agentPhase_Param
		case agentPhase_Param:
			if len(body) < 6 {
				return body, err, false
			}
			req.ParamType = ParamType(binary.LittleEndian.Uint16(body))
			paramLen := binary.LittleEndian.Uint32(body[2:])
			if len(body)-6 < int(paramLen) {
				return body, err, false
			}

			req.Param = make([]byte, paramLen)
			copy(req.Param, body[6:(6 + paramLen)])
			body = body[6+paramLen:]
			return body, nil, true
		}
	}
}
