package server

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"mesh-agent/config"
	"mesh-agent/logger"
)

type HttpContext struct {
	is  InputStream
	req *HttpRequest
}

type HttpRequest struct {
	conn                 Conn
	proto, method        string
	path, query          string
	head, body           string
	remoteAddr           string
	bodyLen              int
	interf               string
	callMethod           string
	parameterTypesString string
	parameter            string

	//profileGetHttpTime   time.Time
	//profileSendAgentTime time.Time
	//profileGetAgentTime  time.Time
	//profileSendHttpTime  time.Time
}

var ParseFormBodyError = errors.New("ParseFormBodyError")

func (hq *HttpRequest) ParseFormBody() error {
	kvs := strings.Split(hq.body, "&")
	if len(kvs) != 4 {
		return ParseFormBodyError
	}
	hq.interf = kvs[0][strings.IndexByte(kvs[0], '=')+1:]
	hq.callMethod = kvs[1][strings.IndexByte(kvs[1], '=')+1:]
	hq.parameterTypesString = kvs[2][strings.IndexByte(kvs[2], '=')+1:]
	hq.parameter = kvs[3][strings.IndexByte(kvs[3], '=')+1:]
	//logger.Info("ParseFormBody", hq.interf, hq.callMethod, hq.parameterTypesString, hq.parameter)
	return nil
}

func (hq *HttpRequest) Response(req *AgentRequest) error {
	// 如果直接打包成了结果，那就直接发送出去
	if req.ParamType == ParamType_Result {
		return hq.conn.Send(req.Param)
	}
	out := AppendResp(nil, "200 OK", "", string(req.Param))
	//logger.Info("Response HttpRequest", string(out))
	return hq.conn.Send(out)
}

// 监听本地20000端口，向consumer服务
func ServeListenHttp(loops int, port int, workerQueues []chan *HttpRequest) error {
	var events Events
	events.NumLoops = loops

	events.Serving = func(srv Server) (action Action) {
		logger.Info("http server started on port %d (loops: %d)", port, srv.NumLoops)
		return
	}

	events.Opened = func(c Conn) (out []byte, opts Options, action Action) {
		if c.GetConnType() == config.ConnTypeHttp {
			c.SetContext(&HttpContext{})
		} else {
			lastCtx := c.Context()
			if lastCtx == nil {
				c.SetContext(&AgentContext{})
			}
			logger.Info("agent opened: laddr: %v: raddr: %v", c.LocalAddr(), c.RemoteAddr())
		}

		opts.ReuseInputBuffer = true
		//logger.Info("http opened: laddr: %v: raddr: %v", c.LocalAddr(), c.RemoteAddr())
		return
	}

	// 被动监听的 close 不用管
	events.Closed = func(c Conn, err error) (action Action) {
		if c.GetConnType() != config.ConnTypeHttp {
			return GlobalRemoteAgentManager.events.Closed(c, err)
		}
		//logger.Info("http closed: %s: %s", c.LocalAddr().String(), c.RemoteAddr().String())
		return
	}

	events.Data = func(c Conn, in []byte) (out []byte, action Action) {
		if c.GetConnType() != config.ConnTypeHttp {
			return GlobalRemoteAgentManager.events.Data(c, in)
		}
		//logger.Info("Data: laddr: %v: raddr: %v, data", c.LocalAddr(), c.RemoteAddr(), string(in))
		if in == nil {
			return
		}
		httpContext := c.Context().(*HttpContext)

		data := httpContext.is.Begin(in)
		// process the pipeline
		for {
			if len(data) > 0 {
				if httpContext.req == nil {
					httpContext.req = &HttpRequest{}
					httpContext.req.conn = c
				}
			} else {
				break
			}
			leftover, err, ready := parseReq(data, httpContext.req)

			if err != nil {
				logger.Warning("bad thing happened\n")
				out = AppendResp(out, "500 Error", "", err.Error()+"\n")
				action = Close
				break
			} else if !ready {
				// request not ready, yet
				data = leftover
				break
			}

			err = httpContext.req.ParseFormBody()
			if err != nil {
				logger.Warning("parse form body error \n")
				out = AppendResp(out, "500 Error", "", err.Error()+"\n")
				action = Close
				break
			}

			//httpContext.req.profileGetHttpTime = time.Now()
			//counterRecvHttp.Incr(1)

			index := getHashCode(httpContext.req.parameter) % *config.ConsumerHttpProcessors
			workerQueues[index] <- httpContext.req
			httpContext.req = nil
			data = leftover
			// handle the request
			//httpContext.req.remoteAddr = c.RemoteAddr().String()
		}
		httpContext.is.End(data)
		return
	}

	// We at least want the single http address.
	addrs := []string{fmt.Sprintf("tcp://:%d?reuseport=true", port)}
	// Start serving!
	ser, err := ServeAndReturn(config.ConnTypeHttp, events, addrs...)
	GlobalRemoteAgentManager.server = ser
	return err
}

func getHashCode(param string) int {
	if len(param) >= 1 {
		return int(param[0])
	}
	return 0
}

var contentLengthHeaderLower = "content-length:"
var contentLengthHeaderUpper = "CONTENT-LENGTH:"
var contentLengthHeaderLen = 15

func getContentLength(header string) (bool, int) {
	headerLen := len(header)
	if headerLen < contentLengthHeaderLen {
		return false, 0
	}
	i := 0
	for ; i < contentLengthHeaderLen; i++ {
		if header[i] != contentLengthHeaderLower[i] &&
			header[i] != contentLengthHeaderUpper[i] {
			return false, 0
		}
	}
	for ; i < headerLen; i++ {
		if header[i] != ' ' {
			l, _ := strconv.Atoi(header[i:])
			return true, l
		}
	}
	return false, 0
}

// AppendResp will append a valid http response to the provide bytes.
// The status param should be the code plus text such as "200 OK".
// The head parameter should be a series of lines ending with "\r\n" or empty.
func AppendResp(b []byte, status, head, body string) []byte {
	b = append(b, "HTTP/1.1"...)
	b = append(b, ' ')
	b = append(b, status...)
	b = append(b, '\r', '\n')
	b = append(b, "Server: nbserver\r\n"...)
	b = append(b, "Connection: keep-alive\r\n"...)
	//b = append(b, "Content-Type: application/json;charset=UTF-8\r\n"...)

	b = append(b, "Date: "...)
	b = time.Now().AppendFormat(b, "Mon, 02 Jan 2006 15:04:05 GMT")
	b = append(b, '\r', '\n')
	if len(body) > 0 {
		b = append(b, "Content-Length: "...)
		b = strconv.AppendInt(b, int64(len(body)), 10)
		b = append(b, '\r', '\n')
	}
	if len(head) > 0 {
		b = append(b, head...)
	}
	b = append(b, '\r', '\n')
	if len(body) > 0 {
		b = append(b, body...)
	}
	return b
}

// parse req is a very simple http request parser. This operation
// waits for the entire payload to be buffered before returning a
// valid request.
func parseReq(data []byte, req *HttpRequest) (leftover []byte, err error, ready bool) {
	sdata := string(data)

	if req.bodyLen > 0 {
		if len(sdata) < req.bodyLen {
			return data, nil, false
		}

		req.body = string(data[0:req.bodyLen])
		return data[req.bodyLen:], nil, true
	}
	var i, s int
	var top string
	var clen int
	var q = -1
	// method, path, proto line
	for ; i < len(sdata); i++ {
		if sdata[i] == ' ' {
			req.method = sdata[s:i]
			for i, s = i+1, i+1; i < len(sdata); i++ {
				if sdata[i] == '?' && q == -1 {
					q = i - s
				} else if sdata[i] == ' ' {
					if q != -1 {
						req.path = sdata[s:q]
						req.query = req.path[q+1 : i]
					} else {
						req.path = sdata[s:i]
					}
					for i, s = i+1, i+1; i < len(sdata); i++ {
						if sdata[i] == '\n' && sdata[i-1] == '\r' {
							req.proto = sdata[s:i]
							i, s = i+1, i+1
							break
						}
					}
					break
				}
			}
			break
		}
	}
	if req.proto == "" {
		return data, nil, false
	}
	top = sdata[:s]
	for ; i < len(sdata); i++ {
		if i > 1 && sdata[i] == '\n' && sdata[i-1] == '\r' {
			line := sdata[s : i-1]
			//fmt.Print("header line %", line, "%\n")
			s = i + 1
			if line == "" {
				req.bodyLen = clen
				req.head = sdata[len(top) : i+1]
				i++
				if clen > 0 {
					if len(sdata[i:]) < clen {
						return data[i:], nil, false
					}
					req.body = sdata[i : i+clen]
					i += clen
				}
				return data[i:], nil, true
			}
			ok, cl := getContentLength(line)
			if ok {
				clen = cl
			}
		}
	}
	// not enough data
	return data, nil, false
}
