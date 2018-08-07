package main

import (
	"flag"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime/pprof"
	"time"

	"mesh-agent/config"
	"mesh-agent/logger"
	"mesh-agent/server"
)

func DumpCpuInfo(seconds int) {
	f, err := os.Create(*config.ProfileDir + *config.Mode + "_cpuprof.prof")
	if err != nil {
		return
	}
	pprof.StartCPUProfile(f)
	time.Sleep((time.Duration)(seconds) * time.Second)
	pprof.StopCPUProfile()
}

func main() {
	rand.Seed(int64(time.Now().UnixNano()))

	flag.Parse()
	//runtime.GOMAXPROCS(*config.MaxProcs)

	//go recordEvent()

	config.ProfileLogger = logger.GetProfileLogger(*config.Mode, *config.ProfileDir)
	config.ProfileGCLogger = logger.GetProfileGCLogger(*config.Mode, *config.ProfileDir)

	if *config.DebugFlag {
		go DumpCpuInfo(180)

		go func() {
			if *config.Mode == "consumer" {
				log.Println(http.ListenAndServe("localhost:8088", nil))
			} else {
				log.Println(http.ListenAndServe("localhost:8089", nil))
			}
		}()
	}

	if *config.Mode == "consumer" {
		server.GlobalRemoteAgentManager.ServeConnectAgent()

		time.Sleep(time.Second)

		server.LocalHttpServer(*config.ConsumerHttpLoops, *config.LocalPort)
		time.Sleep(time.Second)
		go server.GlobalRemoteAgentManager.ListenInterface(server.GlobalInterface)
	} else {

		server.GlobalLocalDubboAgent.ServeConnectDubbo()
		time.Sleep(time.Second)

		server.LocalAgentServer(*config.ProviderAgentLoops, *config.ProviderPort)
		time.Sleep(time.Second)
		go server.GlobalRemoteAgentManager.RegisterInterface(server.GlobalInterface, *config.ProviderPort)
	}
	select {}
}
