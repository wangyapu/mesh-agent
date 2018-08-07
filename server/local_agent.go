package server

import (
	"mesh-agent/config"
)

// provider 监听的本地Agent Server
// 接收到数据后，请求本地的dubbo agent
func LocalAgentServer(loops, port int) {
	workerQueues := make([]chan *AgentRequest, *config.ProviderAgentProcessors)

	for i := 0; i < *config.ProviderAgentProcessors; i++ {
		workerQueues[i] = make(chan *AgentRequest, 100)
		workerQueue := workerQueues[i]

		go func() {
			//runtime.LockOSThread()
			//defer runtime.UnlockOSThread()

			reqList := make([]*AgentRequest, *config.DubboMergeCountMax)
			// 每包不会超过512
			cacheBuf := make([]byte, 0, (1024+256)*(*config.DubboMergeCountMax))
			for req := range workerQueue {
				reqCount := 0
				//queueLen := len(workerQueue)
				//counterProviderAgentQueue.Incr(int64(queueLen))

			ForBlock:
				for {
					reqList[reqCount] = req
					reqCount++
					if reqCount == *config.DubboMergeCountMax {
						break
					}

					select {
					case req = <-workerQueue:
					default:
						break ForBlock
					}

				}

				GlobalLocalDubboAgent.SendDubboRequest(reqList, reqCount, cacheBuf[0:0])
			}
		}()
	}

	ServeListenAgent(loops, port, workerQueues)
}
