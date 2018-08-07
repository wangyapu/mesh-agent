package server

import (
	"flag"
	"time"

	"mesh-agent/ccmap"
	"mesh-agent/config"
	"mesh-agent/logger"
)

var dubboPort = flag.Int("dubbo-port", 20880, "")
var dubboHost = flag.String("dubbo-host", "127.0.0.1", "")
var dubboConnCount = flag.Int("dubbo-conn", 2, "")

var GlobalLocalDubboAgent LocalDubboAgent

type LocalDubboAgent struct {
	requestMap ccmap.ConcurrentMap

	server           *server
	events           *Events
	workerRespQueues []chan *DubboReponse

	connManager ConnectionManager
}

type DubboReponse DubboResponse

type DubboContext struct {
	is InputStream
}

func CreateDubboEvent(loops int, workerQueues []chan *DubboReponse) *Events {
	events := &Events{}
	events.NumLoops = loops
	providerDubboProcessors := int64(*config.ProviderDubboProcessors)

	events.Serving = func(srv Server) (action Action) {
		logger.Info("dubbo agent server started (loops: %d)", srv.NumLoops)
		return
	}

	events.Opened = func(c Conn) (out []byte, opts Options, action Action) {
		lastCtx := c.Context()
		if lastCtx == nil {
			c.SetContext(&DubboContext{})
		}

		opts.ReuseInputBuffer = true

		logger.Info("dubbo agent opened: laddr: %v: raddr: %v", c.LocalAddr(), c.RemoteAddr())
		return
	}

	// producer 向dubbo 发起的链接，在close的时候需要销毁
	// 删除dubbo的连接
	events.Closed = func(c Conn, err error) (action Action) {
		logger.Info("dubbo agent closed: %s: %s", c.LocalAddr(), c.RemoteAddr())
		GlobalLocalDubboAgent.connManager.DeleteConnection(c)
		GlobalLocalDubboAgent.GetConnection()
		return
	}

	events.Data = func(c Conn, in []byte) (out []byte, action Action) {
		if in == nil {
			return
		}
		agentContext := c.Context().(*DubboContext)

		data := agentContext.is.Begin(in)
		// process the pipeline
		for {
			leftover, resp, err := UnpackResponse(data)

			if err != nil {
				if err == ErrHeaderNotEnough {
					// request not ready, yet
					data = leftover
					break
				} else {
					// bad thing happened
					action = Close
					break
				}
			}
			//counterRecvDubbo.Incr(1)
			//AppendRequest(out, httpContext.req)

			response := (*DubboReponse)(resp)
			index := response.ID % providerDubboProcessors
			workerQueues[index] <- response

			// handle the request
			data = leftover
		}
		agentContext.is.End(data)
		return
	}
	return events
}

func (lda *LocalDubboAgent) SendDubboRequest(reqList []*AgentRequest, reqCount int, totalBuf []byte) {
	for i := 0; i < reqCount; i++ {
		req := reqList[i]

		var err error
		//totalBuf, err = dubbojson.PackRequest(totalBuf, &message)
		totalBuf, err = PackRequest(totalBuf, req)
		if err != nil {
			// do something
		}
		lda.requestMap.Set(int(req.RequestID), req)

		//req.profileRemoteAgentSendDubboTime = time.Now()
	}

	lda.GetConnection().Send(totalBuf)
	//counterSendDubbo.Incr(1)
}

func (lda *LocalDubboAgent) ServeConnectDubbo() error {

	lda.requestMap = ccmap.New()
	//lda.workerRespQueue = make(chan *DubboReponse, 1000)
	lda.workerRespQueues = make([]chan *DubboReponse, *config.ProviderDubboProcessors)

	for i := 0; i < *config.ProviderDubboProcessors; i++ {
		lda.workerRespQueues[i] = make(chan *DubboReponse, 100)
		workerRespQueue := lda.workerRespQueues[i]

		go func() {
			//runtime.LockOSThread()
			//defer runtime.UnlockOSThread()

			cachedBuf := make([]byte, 2048*(*config.AgentMergeCountMax))
			for resp := range workerRespQueue {
				//counterDubboQueue.Incr(int64(len(lda.workerRespQueue)))

				respCount := 0
				totalBuf := cachedBuf[0:0]
				var conn Conn

			ForBlock:
				for {

					respId := int(resp.ID)
					obj, ok := GlobalLocalDubboAgent.requestMap.Get(respId)

					//obj, ok := GlobalLocalDubboAgent.requestMap.Load(uint64(resp.ID))
					if !ok {
						logger.Info("receive dubbo client's response, but no this req id %d", respId)
						break
					}
					agentReq := obj.(*AgentRequest)
					if !resp.Status.OK() {
						logger.Info("receive dubbo client's response, but status not OK ", int(resp.ID), resp.Status.String())
						buf := make([]byte, 0, 2048)
						GlobalLocalDubboAgent.SendDubboRequest([]*AgentRequest{agentReq}, 1, buf)
						break
					}

					conn = agentReq.conn
					if true {
						// 直接打包成http
						totalBuf, _ = PackAgentRequest(totalBuf, 200, agentReq.RequestID, "", "", ParamType_Result, AppendResp(nil, "200 OK", "", string(resp.Data)))
					} else {
						totalBuf, _ = PackAgentRequest(totalBuf, 200, agentReq.RequestID, agentReq.Interf, agentReq.Method, agentReq.ParamType, resp.Data)
					}

					GlobalLocalDubboAgent.requestMap.Remove(respId)

					//agentReq.profileRemoteAgentSendAgentTime = time.Now()
					//counterProviderTotalLatency.Incr(int64(agentReq.profileRemoteAgentSendAgentTime.Sub(agentReq.profileRemoteAgentGetTime)))
					//counterProviderDubboLatency.Incr(int64(agentReq.profileRemoteAgentSendAgentTime.Sub(agentReq.profileRemoteAgentSendDubboTime)))

					respCount++
					if respCount == *config.AgentMergeCountMax {
						break
					}

					select {
					case resp = <-workerRespQueue:
					default:
						break ForBlock
					}

				}

				//counterSendConsumer.Incr(1)
				if len(totalBuf) > 0 && conn != nil {
					conn.Send(totalBuf)
				}

				//ProfileLogger.Info(agentReq.profileRemoteAgentSendAgentTime.Sub(agentReq.profileRemoteAgentGetTime),
				//	agentReq.profileRemoteAgentGetDubboTime.Sub(agentReq.profileRemoteAgentSendDubboTime),
				//	agentReq.profileRemoteAgentSendAgentTime.Sub(agentReq.profileRemoteAgentSendDubboTime),
				//	agentReq.profileRemoteAgentSendAgentTime.Sub(agentReq.profileRemoteAgentGetTime)-agentReq.profileRemoteAgentGetDubboTime.Sub(agentReq.profileRemoteAgentSendDubboTime))
			}
		}()
	}

	//events := CreateDubboEvent(loops, lda.workerRespQueue)
	events := CreateDubboEvent(0, lda.workerRespQueues)
	var err error
	//lda.server, err = ConnServe(*events)
	lda.events = events
	return err
}

func (lda *LocalDubboAgent) GetConnection() Conn {
	resultConn, connCount := lda.connManager.GetConnection()
	createConnCount := *dubboConnCount - connCount

	makeConn := func() (Conn, error) {
		conn, err := outConnect(lda.server, *dubboHost, *dubboPort, config.ConnTypeDubbo, &DubboContext{})
		if err != nil {
			logger.Warning("CONNECT_DUBBO_ERROR", err, *dubboPort, *dubboPort)
			return nil, err
		}

		lda.connManager.AddConnection(conn)
		return conn, nil
	}

	if connCount == 0 {
		for true {
			conn, err := makeConn()
			if err == nil {
				resultConn = conn
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		createConnCount--
	}

	for createConnCount > 0 {
		go makeConn()
		createConnCount--
	}

	return resultConn.(Conn)
}
