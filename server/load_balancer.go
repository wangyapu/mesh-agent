package server

import "sync/atomic"

type LoadBalancer struct {
	weight    []uint32
	minWeight []uint32

	totalCount uint32
	nowBase    uint32

	lastCount     uint32
	lastIndex     uint32
	lastBaseCount uint32
}

func (lb *LoadBalancer) Get() int {
	totalCount := atomic.LoadUint32(&lb.totalCount)
	if totalCount == 0 {
		return int(totalCount)
	}
	newCount := atomic.AddUint32(&lb.lastCount, 1)
	if newCount-atomic.LoadUint32(&lb.lastBaseCount) >
		lb.minWeight[atomic.LoadUint32(&lb.lastIndex)%totalCount] {
		newIndex := atomic.AddUint32(&lb.lastIndex, 1)
		atomic.StoreUint32(&lb.lastBaseCount, newCount)
		return int(newIndex % totalCount)
	}
	return int(atomic.LoadUint32(&lb.lastIndex) % totalCount)
}

func (lb *LoadBalancer) Update(index, weight uint32) {
	if len(lb.weight) <= int(index) {
		atomic.AddUint32(&lb.totalCount, 1)
		if index == 0 {
			lb.minWeight = append(lb.minWeight, weight/100)
			lb.nowBase = 100
		} else {
			lb.minWeight = append(lb.minWeight, weight/lb.nowBase)
		}
		lb.weight = append(lb.weight, weight)
	} else {
		atomic.StoreUint32(&lb.weight[index], weight)
		atomic.StoreUint32(&lb.minWeight[index], weight/lb.nowBase)
	}
}

func (lb *LoadBalancer) Delete(index uint32) {
	if len(lb.weight) <= int(index) {
		return
	}
	// dec
	lb.minWeight = append(lb.minWeight[0:index], lb.minWeight[index:]...)
	lb.weight = append(lb.weight[0:index], lb.weight[index+1:]...)
	atomic.AddUint32(&lb.totalCount, ^uint32(0))
}
