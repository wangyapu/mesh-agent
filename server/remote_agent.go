package server

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"mesh-agent/ccmap"
	"mesh-agent/config"
	"mesh-agent/logger"
	"mesh-agent/util"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
)

var etcdHost = flag.String("etcd-host", "localhost", "")
var etcdPort = flag.Int("etcd-port", 2379, "")

var noAgentConnectionError = errors.New("noAgentConnectionError")
var noAgentError = errors.New("noAgentError")

var GlobalInterface = "/dubbomesh/com.alibaba.dubbo.performance.demo.provider.IHelloService/"

var GlobalRemoteAgentManager RemoteAgentManager

type RemoteAgent struct {
	connManager  ConnectionManager
	lastConn     Conn
	sendCount    uint64
	addr         string
	port         int
	allInterface sync.Map
	requestMap   ccmap.ConcurrentMap
	reqID        uint64
	defaultConn  int
}

func (ra *RemoteAgent) GetConnection() Conn {
	if ra.lastConn != nil {
		return ra.lastConn
	}
	connInterf, connCount := ra.connManager.GetConnection()
	createConnCount := ra.defaultConn - connCount
	for createConnCount > 0 {
		go ra.CreateConnection(nil)
		createConnCount--
	}
	if connInterf != nil {
		return connInterf.(Conn)
	}
	conn, _ := ra.CreateConnection(nil)
	ra.lastConn = conn
	return conn
}

func (ra *RemoteAgent) AddInterface(interf string) {
	ra.allInterface.Store(interf, nil)
}

func (ra *RemoteAgent) CreateConnection(wg *sync.WaitGroup) (Conn, error) {
	conn, err := outConnect(GlobalRemoteAgentManager.server, ra.addr, ra.port, config.ConnTypeAgent, &AgentContext{ra: ra})
	if err == nil {
		ra.connManager.AddConnection(conn)
	}
	return conn, err
}

func (ra *RemoteAgent) SendRequest(reqList []*AgentRequest, httpReqList []*HttpRequest, reqCount int, totalBuf []byte) error {
	conn := ra.GetConnection()
	if conn == nil {
		return noAgentConnectionError
	}
	for i := 0; i < reqCount; i++ {
		httpReq := httpReqList[i]
		req := reqList[i]

		req.RequestID = atomic.AddUint64(&ra.reqID, 1)
		//ra.requestMap.Store(req.RequestID, httpReq)

		ra.requestMap.Set(int(req.RequestID), httpReq)

		totalBuf, _ = PackAgentRequest(totalBuf, req.Result, req.RequestID, req.Interf, req.Method, req.ParamType, req.Param)

		//httpReq.profileSendAgentTime = time.Now()

	}
	//counterSendProvider.Incr(1)
	return conn.Send(totalBuf)
}

type RemoteAgentManager struct {
	allAgents         sync.Map // key addr, value *RemoteAgent
	cacheInterfaceMap sync.Map // key interface, val []RemoteAgent

	connectionManager ConnectionManager
	workerRespQueues  []chan *AgentRequest
	server            *server
	events            *Events
}

func (ram *RemoteAgentManager) AddAgent(addr string, port int, interf string, defaultConn int, weight int) error {
	existIdx := -1
	allConns := ram.connectionManager.GetAllConnections()

	for idx, conn := range allConns {
		agent := conn.(*RemoteAgent)

		if agent.addr == addr && agent.port == port {
			existIdx = idx
			break
		}
	}

	if existIdx >= 0 {
		if weight == 0 {
			return nil
		}
		logger.Info("update agent weight %s %s %d", addr, port, weight)
		ram.connectionManager.UpdateLB(uint32(existIdx), uint32(weight))
		return nil
	}
	existIdx = len(allConns)
	logger.Info("create agent weight %s %s %d", addr, port, weight)
	ra := &RemoteAgent{
		addr:        addr,
		port:        port,
		defaultConn: defaultConn,
		requestMap:  ccmap.New(),
	}
	ra.AddInterface(interf)
	successCount := 0

	for i := 0; i < defaultConn; i++ {
		_, err := ra.CreateConnection(nil)
		if err == nil {
			successCount++
		} else {
			logger.Info("create agent weight %s %s %d", addr, port, weight)
		}
	}
	ram.connectionManager.AddConnection(ra)
	ram.connectionManager.UpdateLB(uint32(existIdx), uint32(weight))

	return nil
}

func (ram *RemoteAgentManager) getInterfaceKey(interf string, port int) string {
	return interf + util.GetHostName() + ":" + strconv.Itoa(port)
}

func (ram *RemoteAgentManager) ForwardRequest(agentReqList []*AgentRequest, httpReqList []*HttpRequest, reqCount int, totalBuf []byte) error {
	conn, connCount := ram.connectionManager.GetConnection()
	if connCount == 0 {
		logger.Info("empty agents \n")
		return noAgentError
	}

	//httpReq.profileGetAgentTime = agentReq.profileRemoteAgentGetTime
	return conn.(*RemoteAgent).SendRequest(agentReqList, httpReqList, reqCount, totalBuf)
}

// 获取当前能力值，基础值为1000
func (ram *RemoteAgentManager) GetLBWeight() string {
	if *config.ProviderWeight > 0 {
		return strconv.Itoa(*config.ProviderWeight)
	}
	//hits := counterProviderTotalLatency.Hits()
	//rate := counterProviderTotalLatency.Rate() / float64(time.Microsecond)

	//if hits > 10 && rate > float64(30000) {
	//	// 50 - 100，超过60为100，50为1100
	//	weight := 100 * (60 - rate/1000)
	//	if weight <= 0 {
	//		weight = 100
	//	} else {
	//		weight += 100
	//	}
	//	return strconv.Itoa(int(weight))
	//}
	return "0"
}

func (ram *RemoteAgentManager) RegisterInterface(interf string, port int) {
	cli, err := etcd.New(etcd.Config{
		Endpoints:   []string{fmt.Sprintf("http://%s:%d", *etcdHost, *etcdPort)},
		DialTimeout: time.Second * 3,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	resp, err := cli.Grant(context.TODO(), 5)
	if err != nil {
		log.Fatal(err)
	}

	interfaceKey := ram.getInterfaceKey(interf, port)
	logger.Info("register interface to etcd", interfaceKey)
	_, err = cli.Put(context.TODO(), interfaceKey, "", etcd.WithLease(resp.ID))
	if err != nil {
		log.Fatal(err)
	}

	// the key will be kept forever
	ch, kaerr := cli.KeepAlive(context.TODO(), resp.ID)
	if kaerr != nil {
		log.Fatal(kaerr)
	}
	for {
		time.Sleep(time.Second * 10)
		cli.Put(context.TODO(), interfaceKey, ram.GetLBWeight(), etcd.WithLease(resp.ID))
	}
	ka := <-ch
	logger.Info("ttl:", ka.TTL)
}

func (ram *RemoteAgentManager) DeleteRemoteAgentConnection(conn Conn) {
	agentCtx := conn.Context().(*AgentContext)
	deletedCount := agentCtx.ra.connManager.DeleteConnection(conn)

	agentCtx.ra.CreateConnection(nil)
	logger.Info("ServeConnectAgent agent closed", conn.LocalAddr(), conn.RemoteAddr(), "deleted count", deletedCount, "now count", agentCtx.ra.connManager.GetConnectionCount())
}

func (ram *RemoteAgentManager) ServeConnectAgent() error {
	//ram.workerRespQueue = make(chan *AgentRequest, 1000)
	ram.workerRespQueues = make([]chan *AgentRequest, *config.ConsumerAgentProcessors)

	for i := 0; i < *config.ConsumerAgentProcessors; i++ {
		ram.workerRespQueues[i] = make(chan *AgentRequest, 100)
		workerRespQueue := ram.workerRespQueues[i]

		go func() {
			//runtime.LockOSThread()
			//defer runtime.UnlockOSThread()

			for resp := range workerRespQueue {
				//counterConsumerAgentQueue.Incr(int64(len(ram.workerRespQueue)))
				ctx := resp.conn.Context().(*AgentContext)

				respId := int(resp.RequestID)
				obj, ok := ctx.ra.requestMap.Get(respId)

				if !ok {
					logger.Info("receive remote agent's response, but no this req id %d", respId)
					continue
				}

				httpReq := obj.(*HttpRequest)
				httpReq.Response(resp)

				//httpReq.profileGetAgentTime = time.Now()
				//counterSendHttp.Incr(1)
				//httpReq.profileSendHttpTime = time.Now()

				//counterConsumerRPCLatency.Incr(int64(httpReq.profileGetAgentTime.Sub(httpReq.profileSendAgentTime)))
				//counterConsumerTotalLatency.Incr(int64(httpReq.profileSendHttpTime.Sub(httpReq.profileGetHttpTime)))
				//ProfileLogger.Info(httpReq.profileSendHttpTime.Sub(httpReq.profileGetHttpTime),
				//	httpReq.profileGetAgentTime.Sub(httpReq.profileSendAgentTime),
				//	httpReq.profileSendHttpTime.Sub(httpReq.profileSendAgentTime),
				//	httpReq.profileSendHttpTime.Sub(httpReq.profileGetHttpTime)-httpReq.profileGetAgentTime.Sub(httpReq.profileSendAgentTime))

				//ctx.ra.requestMap.Delete(resp.RequestID)

				ctx.ra.requestMap.Remove(respId)
			}
		}()
	}

	events := CreateAgentEvent(0, ram.workerRespQueues, uint64(*config.ConsumerAgentProcessors))
	events.Closed = func(c Conn, err error) (action Action) {
		ram.DeleteRemoteAgentConnection(c)

		return
	}
	var err error
	ram.events = events
	//ram.server, err = ConnServe(*events)
	return err
}

func (ram *RemoteAgentManager) ListenInterface(interf string) {
	cfg := etcd.Config{
		Endpoints:   []string{fmt.Sprintf("http://%s:%d", *etcdHost, *etcdPort)},
		DialTimeout: time.Second * 3,
	}
	c, err := etcd.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	defer c.Close()
	if resp, err := c.Get(context.Background(), interf, etcd.WithPrefix()); err != nil {
		log.Fatal(err)
	} else {
		for _, ev := range resp.Kvs {
			logger.Info(fmt.Sprintf("etcd key events : ", ev.Key, string(ev.Key), string(ev.Value)))
			keyTotal := string(ev.Key)
			tail := keyTotal[strings.LastIndexByte(keyTotal, '/')+1:]
			keyValuePairs := strings.Split(tail, ":")
			addr := keyValuePairs[0]
			port, _ := strconv.Atoi(keyValuePairs[1])
			ram.AddAgent(addr, port, interf, *config.DefaultAgentCount, 1000)
		}
	}

	rch := c.Watch(context.Background(), interf, etcd.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			switch ev.Type {
			case mvccpb.PUT:
				keyTotal := string(ev.Kv.Key)
				tail := keyTotal[strings.LastIndexByte(keyTotal, '/')+1:]
				keyValuePairs := strings.Split(tail, ":")
				addr := keyValuePairs[0]
				port, _ := strconv.Atoi(keyValuePairs[1])
				weight, _ := strconv.Atoi(string(ev.Kv.Value))
				if weight > 0 {
					// @todo 更新Agent 能力
					ram.AddAgent(addr, port, interf, 4, weight)
				} else {
					ram.AddAgent(addr, port, interf, 4, 0)
				}
			case mvccpb.DELETE:
				// @todo 删除Agent
			}
		}
	}
}
