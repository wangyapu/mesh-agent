package server

//import (
//	"fmt"
//	"runtime"
//	"runtime/debug"
//	"strconv"
//	"time"
//
//	"github.com/paulbellamy/ratecounter"
//	"mesh-agent/config"
//)
//
//var counterRecvHttp *ratecounter.RateCounter
//var counterRecvProvider *ratecounter.RateCounter
//var counterRecvConsumer *ratecounter.RateCounter
//var counterRecvDubbo *ratecounter.RateCounter
//
//var counterSendHttp *ratecounter.RateCounter
//var counterSendProvider *ratecounter.RateCounter
//var counterSendConsumer *ratecounter.RateCounter
//var counterSendDubbo *ratecounter.RateCounter
//
//var counterHttpQueue *ratecounter.AvgRateCounter
//var counterProviderAgentQueue *ratecounter.AvgRateCounter
//var counterConsumerAgentQueue *ratecounter.AvgRateCounter
//var counterDubboQueue *ratecounter.AvgRateCounter
//
//var counterProviderDubboLatency *ratecounter.AvgRateCounter
//var counterProviderTotalLatency *ratecounter.AvgRateCounter
//var counterConsumerRPCLatency *ratecounter.AvgRateCounter
//var counterConsumerTotalLatency *ratecounter.AvgRateCounter
//
//func recordEvent() {
//	for {
//
//		time.Sleep(time.Second)
//		if *config.Mode == "consumer" {
//			config.ProfileLogger.Info("counterRecvHttp  ", counterRecvHttp.Rate(),
//				" counterSendHttp  ", counterSendHttp.Rate(),
//				" counterRecvProvider  ", counterRecvProvider.Rate(),
//				" counterSendProvider  ", counterSendProvider.Rate())
//
//			config.ProfileLogger.Info("counterConsumerAgentQueue  ", strconv.FormatFloat(counterConsumerAgentQueue.Rate(), 'f', 5, 64))
//			config.ProfileLogger.Info("counterHttpQueue  ", strconv.FormatFloat(counterHttpQueue.Rate(), 'f', 5, 64))
//			config.ProfileLogger.Info("counterConsumerRPCLatency  ", strconv.FormatFloat(counterConsumerRPCLatency.Rate()/float64(time.Microsecond), 'f', 5, 64))
//			config.ProfileLogger.Info("counterConsumerTotalLatency  ", strconv.FormatFloat(counterConsumerTotalLatency.Rate()/float64(time.Microsecond), 'f', 5, 64))
//		} else {
//			config.ProfileLogger.Info("counterRecvConsumer  ", counterRecvConsumer.Rate(),
//				" counterSendConsumer  ", counterSendConsumer.Rate(),
//				" counterRecvDubbo  ", counterRecvDubbo.Rate(),
//				" counterSendConsumer  ", counterSendConsumer.Rate())
//
//			config.ProfileLogger.Info("counterProviderAgentQueue  ", strconv.FormatFloat(counterProviderAgentQueue.Rate(), 'f', 5, 64))
//			config.ProfileLogger.Info("counterDubboQueue  ", strconv.FormatFloat(counterDubboQueue.Rate(), 'f', 5, 64))
//			config.ProfileLogger.Info("counterProviderDubboLatency  ", strconv.FormatFloat(counterProviderDubboLatency.Rate()/float64(time.Microsecond), 'f', 5, 64))
//			config.ProfileLogger.Info("counterProviderTotalLatency  ", strconv.FormatFloat(counterProviderTotalLatency.Rate()/float64(time.Microsecond), 'f', 5, 64))
//		}
//		if *config.DebugFlag {
//			gcstats := &debug.GCStats{PauseQuantiles: make([]time.Duration, 5)}
//			debug.ReadGCStats(gcstats)
//			memStats := &runtime.MemStats{}
//			runtime.ReadMemStats(memStats)
//			config.ProfileGCLogger.Info("gc status", fmt.Sprintf("%+v", *gcstats), "mem status", fmt.Sprintf("%+v", *memStats))
//		}
//
//	}
//}
//
//func init() {
//	counterRecvHttp = ratecounter.NewRateCounter(time.Second)
//	counterRecvProvider = ratecounter.NewRateCounter(time.Second)
//	counterRecvConsumer = ratecounter.NewRateCounter(time.Second)
//	counterRecvDubbo = ratecounter.NewRateCounter(time.Second)
//
//	counterSendProvider = ratecounter.NewRateCounter(time.Second)
//	counterSendConsumer = ratecounter.NewRateCounter(time.Second)
//	counterSendHttp = ratecounter.NewRateCounter(time.Second)
//	counterSendDubbo = ratecounter.NewRateCounter(time.Second)
//
//	counterHttpQueue = ratecounter.NewAvgRateCounter(time.Second)
//	counterConsumerAgentQueue = ratecounter.NewAvgRateCounter(time.Second)
//	counterDubboQueue = ratecounter.NewAvgRateCounter(time.Second)
//	counterProviderAgentQueue = ratecounter.NewAvgRateCounter(time.Second)
//
//	counterConsumerTotalLatency = ratecounter.NewAvgRateCounter(time.Second)
//	counterConsumerRPCLatency = ratecounter.NewAvgRateCounter(time.Second)
//	// 调整到一个小时，只负责计算平均值就好了
//	counterProviderTotalLatency = ratecounter.NewAvgRateCounter(time.Hour)
//	counterProviderDubboLatency = ratecounter.NewAvgRateCounter(time.Second)
//
//}
