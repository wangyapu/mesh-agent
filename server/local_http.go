package server

import (
	"mesh-agent/config"
	"mesh-agent/logger"
)

func LocalHttpServer(loops, port int) {
	workerQueues := make([]chan *HttpRequest, *config.ConsumerHttpProcessors)

	for i := 0; i < *config.ConsumerHttpProcessors; i++ {
		workerQueues[i] = make(chan *HttpRequest, 100)
		workerQueue := workerQueues[i]

		go func() {
			//runtime.LockOSThread()
			//defer runtime.UnlockOSThread()

			cachedBuf := make([]byte, (128+1024)*(*config.HttpMergeCountMax))
			httpReqList := make([]*HttpRequest, *config.HttpMergeCountMax)
			agentReqList := make([]*AgentRequest, *config.HttpMergeCountMax)
			for req := range workerQueue {
				reqCount := 0
				//counterHttpQueue.Incr(int64(len(workerQueue)))
			ForBlock:
				for {
					httpReqList[reqCount] = req
					agentReqList[reqCount] = &AgentRequest{
						Interf:    req.interf,
						Method:    req.callMethod,
						ParamType: ParamType_String,
						Param:     []byte(req.parameter),
					}

					reqCount++

					if reqCount == *config.HttpMergeCountMax {
						break
					}

					select {
					case req = <-workerQueue:
					default:
						break ForBlock
					}
				}

				if err := GlobalRemoteAgentManager.ForwardRequest(agentReqList, httpReqList, reqCount, cachedBuf[0:0]); err != nil {
					logger.Warning("forward request error", err)
					req.conn.Send(AppendResp(nil, "500", err.Error(), "forward request error"))
				}
			}
		}()
	}

	//ServeListenHttp(loops, port, workerQueue)
	ServeListenHttp(loops, port, workerQueues)
}
