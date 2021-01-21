package main

import (
	"os"
	"time"

	"charlie/i0.0.2/cls"
)

var lprintf func(int, string, ...interface{}) = cls.Lprintf
var StatTime, NodeTime, SvcTime, DataTime int
var R Router
var NotUseNodes string

func sub_main() {

	fname := cls.Cls_conf(os.Args)
	lprintf(4, "[INFO] smartrouter start \n")

	// app conf
	if getConf(fname) < 0 {
		return
	}

	// recieve node, svc data
	for {
		if getData() > 0 {
			break
		}

		lprintf(1, "[FAIL] sphere get Data fail \n")
		time.Sleep(5 * time.Second)
	}

	f := make(chan *reqNode)
	//go rvLSA(R.webPort, f) // listen lsa packet
	go routerTcp(cls.ListenIP, R.webPort, f) // listen lsa packet
	go fHandler(f)                           // flooding

	// get service cost
	go svcCheck(R.svcTime)

	// get node cost
	go nodeCheck(R.nodeTime)

	// requset lsa data to another node
	go statCheck(R.statTime)

	// get notify to agent
	go notifyRead(R.fr, R.fe, R.ft)

	// svc on
	for _, si := range SvcRoute.m {

		//go routerHttp(R.listenIp, si)

		//go router("127.0.0.4", val)

		if si.protocol == HTTP || si.protocol == HTTPS {
			go routerHttp(R.listenIp, si)
		} else {
			go routerTcp(R.listenIp, si.lPort, nil)
		}
		/* else if si.protocol == HTTPS {
			go routerHttps(R.listenIp, si)
		}
		*/

	}

	// recieve node, svc data
	DataTime = int(time.Now().Unix())
	for {
		nowTime := int(time.Now().Unix())

		if nowTime-DataTime >= R.reportTime {
			getData()
			DataTime = nowTime
		}

		time.Sleep(3 * time.Second)
	}
}
