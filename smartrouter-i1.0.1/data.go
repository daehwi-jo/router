package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"strconv"
	"time"

	"charlie/i0.0.2/cls"
)

func getConf(fname string) int {

	var err error

	v, r := cls.GetTokenValue("STAT_TIME", fname)
	if r != cls.CONF_ERR {
		R.statTime, err = strconv.Atoi(v)
		if err != nil {
			lprintf(1, "[ERROR] STAT_TIME error(%s) \n", err.Error())
			return ERROR
		}
	} else {
		R.statTime = 180
	}

	v, r = cls.GetTokenValue("SERVICE_TIME", fname)
	if r != cls.CONF_ERR {
		R.svcTime, err = strconv.Atoi(v)
		if err != nil {
			lprintf(1, "[ERROR] SERVICE_TIME error(%s) \n", err.Error())
			return ERROR
		}
	} else {
		R.svcTime = 120
	}

	v, r = cls.GetTokenValue("CONNECT_TIME", fname)
	if r != cls.CONF_ERR {
		R.conTime, err = strconv.Atoi(v)
		if err != nil {
			lprintf(1, "[ERROR] CONNECT_TIME error(%s) \n", err.Error())
			return ERROR
		}
	} else {
		R.conTime = 5
	}

	v, r = cls.GetTokenValue("NODE_TIME", fname)
	if r != cls.CONF_ERR {
		R.nodeTime, err = strconv.Atoi(v)
		if err != nil {
			lprintf(1, "[ERROR] NODE_TIME error(%s) \n", err.Error())
			return ERROR
		}
	} else {
		R.nodeTime = 120
	}

	v, r = cls.GetTokenValue("REPORT_TIME", fname)
	if r != cls.CONF_ERR {
		R.reportTime, err = strconv.Atoi(v)
		if err != nil {
			lprintf(1, "[ERROR] REPORT_TIME error(%s) \n", err.Error())
			return ERROR
		}
	} else {
		R.reportTime = 30
	}

	v, r = cls.GetTokenValue("LISTENIP", fname)
	if r != cls.CONF_ERR {
		R.listenIp = v
	} else {
		R.listenIp = "127.0.0.4"
	}

	v, r = cls.GetTokenValue("WEB_PORT", fname)
	if r != cls.CONF_ERR {
		R.webPort = v
	} else {
		lprintf(1, "[FAIL] not found WEB_PORT in conf(%s) \n", fname)
		return ERROR
	}

	v, r = cls.GetTokenValue("AGENT_PORT", fname)
	if r != cls.CONF_ERR {
		R.agentPort = v
	} else {
		lprintf(1, "[FAIL] not found AGENT_PORT in conf(%s) \n", fname)
		return ERROR
	}

	v, r = cls.GetTokenValue("FEXIST", fname)
	if r != cls.CONF_ERR {
		R.fe = v
	} else {
		R.fe = "/smartagent/Plugins/DFA/smartagent/tmp/smartrouter/scale.ctl"
	}

	v, r = cls.GetTokenValue("FREAD", fname)
	if r != cls.CONF_ERR {
		R.fr = v
	} else {
		R.fr = "/smartagent/Plugins/DFA/smartagent/tmp/smartrouter/scale.data"
	}

	v, r = cls.GetTokenValue("FTIME", fname)
	if r != cls.CONF_ERR {
		R.ft, err = strconv.Atoi(v)
		if err != nil {
			if err != nil {
				lprintf(1, "[ERROR] FTIME error(%s) \n", err.Error())
				return ERROR
			}
		}
	} else {
		R.ft = 3
	}

	v, r = cls.GetTokenValue("CERT_PATH", fname)
	if r != cls.CONF_ERR {
		R.cp = v
	} else {
		R.cp = "/smartagent/Plugins/DFA/smartagent/cert"
	}

	lprintf(4, "[INFO] service time(%d) \n", R.svcTime)
	lprintf(4, "[INFO] node time(%d) \n", R.nodeTime)
	lprintf(4, "[INFO] stat time(%d) \n", R.statTime)
	lprintf(4, "[INFO] report time(%d) \n", R.reportTime)
	lprintf(4, "[INFO] connect timeout(%d) \n", R.conTime)
	lprintf(4, "[INFO] router ip(%s) \n", R.listenIp)
	lprintf(4, "[INFO] webPort(%s) \n", R.webPort)
	lprintf(4, "[INFO] agentPort(%s) \n", R.agentPort)

	return SUCCESS
}

// to sphere
func getData() int {

	// test data
	/*
		if testData() < 0 {
			return ERROR
		}

		return SUCCESS
	*/
	var req ReportMsg
	req.RouterID = R.id
	req.Version = R.version

	SvcRoute.Lock()
	for _, si := range SvcRoute.m {

		slen := len(si.nRank)
		if slen == 0 {
			continue
		}

		var p PathInfo
		p.ServiceIp = si.tServer
		p.NodeInfo = si.nRank[:slen-1]

		req.ShortPath = append(req.ShortPath, p)

	}
	SvcRoute.Unlock()

	LSDB.Lock()
	rlsa, exists := LSDB.m[R.nodeIp]
	if exists {
		for serverIp, cost := range rlsa {
			var sc ServerCost
			sc.ServerIp = serverIp
			sc.Cost = cost

			req.ServerCosts = append(req.ServerCosts, sc)
		}
	}
	LSDB.Unlock()

	lprintf(4, "[INFO] router Request %v \n", req)

	jsonBytes, err := json.Marshal(req)
	if err != nil {
		lprintf(1, "[ERROR] json marshal fail(%s)", err.Error())
		return ERROR
	}

	resp, err := cls.HttpSendJSON(cls.TCP_SPHERE, "GET", "router/report", jsonBytes, true)
	if err != nil {
		lprintf(1, "[ERROR] cls HttpSendRequest fail, error(%s) \n", err.Error())
		return ERROR
	}

	if resp.StatusCode != 200 {
		lprintf(1, "[ERROR] cls HttpSendRequest fail, resp code(%d)\n", resp.StatusCode)
		return ERROR
	}

	if setData(resp) < 0 {
		return ERROR
	}

	return SUCCESS
}

func setData(resp *http.Response) int {

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		lprintf(1, "[ERROR] response body read err(%s) \n", err.Error())
		return ERROR
	}
	defer resp.Body.Close()

	if len(string(data)) == 0 || string(data) == "OK" {
		return SUCCESS
	}

	respMsg, rst := parseResp(data)
	if rst < 0 {
		lprintf(1, "[ERROR] response data parsing err \n")
		return ERROR
	}

	printRespMsg(respMsg)

	if R.id != respMsg.RouterID {
		lprintf(4, "[INFO] router id(%d) response router id(%d) \n", R.id, respMsg.RouterID)
		R.id = respMsg.RouterID
	}

	if R.version != respMsg.Version {
		lprintf(4, "[INFO] router v(%s) response router v(%s) \n", R.version, respMsg.Version)
		R.version = respMsg.Version
	}

	pIp := getPublic(R.agentPort)

	if R.nodeIp != pIp {
		lprintf(4, "[INFO] router ip(%s) response router ip(%s) \n", R.nodeIp, pIp)
		R.nodeIp = pIp
	}

	for i := 0; i < len(respMsg.ServiceList); i++ {

		rs := respMsg.ServiceList[i]

		var key string

		if rs.Protocol == HTTP || rs.Protocol == HTTPS {
			key = rs.Fqdn
		} else if rs.Protocol == TCP {
			key = rs.LPort
		}

		// svc map
		sv, _ := SvcRoute.m[key]

		sv.tServer = rs.ServiceIP
		sv.fqdn = rs.Fqdn
		sv.domain = rs.Domain
		sv.protocol = rs.Protocol
		sv.lPort = rs.LPort
		sv.tPort = rs.TPort
		sv.privateKey = rs.PrivateKey
		sv.publicKey = rs.PublicKey

		SvcRoute.m[key] = sv

		//lprintf(4, "[INFO] get shere svc data(%s) \n", sv.tServer)

		// router lsa
		rlsa, exists := LSDB.m[R.nodeIp]
		if !exists {
			rlsa = make(map[string]int)
		}

		_, exists = rlsa[sv.tServer]
		if !exists {
			rlsa[sv.tServer] = MAXCOST
		}

		// svc lsa
		lsa, exists := LSDB.m[sv.tServer]
		if !exists {
			lsa = make(map[string]int)
		}

		NotUseNodes = ""

		for j := 0; j < len(rs.NodeList); j++ {

			_, exists = lsa[rs.NodeList[j].NodeIp]
			if !exists {
				lsa[rs.NodeList[j].NodeIp] = MAXCOST // 초기값
			}

			// router lsa
			_, exists = rlsa[rs.NodeList[j].NodeIp]
			if !exists && rs.NodeList[j].NodeIp != R.nodeIp {
				rlsa[rs.NodeList[j].NodeIp] = MAXCOST
			}

			// 경로제어에서 장비 미 사용
			if rs.NodeList[j].NodeUse == 0 {
				NotUseNodes += rs.NodeList[j].NodeIp + " "
			}
		}

		LSDB.m[sv.tServer] = lsa
		LSDB.m[R.nodeIp] = rlsa
	}

	return SUCCESS
}

/*
func testData() int {

	var respMsg ResponseMsg
	respMsg.RouterID = 1
	respMsg.Version = "2"

	var svcList SL
	svcList.Fqdn = "www.test1.com"
	svcList.LPort = "80"
	svcList.TPort = "80"
	svcList.ServiceIP = "192.168.41.49"
	svcList.Protocol = HTTP

	/*
		for i := 99; i < 109; i++ {
			ipaddr := fmt.Sprintf("192.168.41.%d", i)
			svcList.NodeList = append(svcList.NodeList, ipaddr)
		}


	ipaddr := "192.168.30.127"
	svcList.NodeList = append(svcList.NodeList, ipaddr)
	ipaddr = "192.168.41.99"
	svcList.NodeList = append(svcList.NodeList, ipaddr)

	respMsg.ServiceList = append(respMsg.ServiceList, svcList)

	if R.id != respMsg.RouterID {
		lprintf(4, "[INFO] router id(%d) response router id(%d) \n", R.id, respMsg.RouterID)
		R.id = respMsg.RouterID
	}

	if R.version != respMsg.Version {
		lprintf(4, "[INFO] router v(%s) response router v(%s) \n", R.version, respMsg.Version)
		R.version = respMsg.Version
	}

	for i := 0; i < len(respMsg.ServiceList); i++ {

		rs := respMsg.ServiceList[i]

		var key string

		if rs.Protocol == HTTP {
			key = rs.Fqdn
		} else if rs.Protocol == TCP {
			key = rs.LPort
		}

		// svc map
		sv, _ := SvcRoute.m[key]

		sv.tServer = rs.ServiceIP
		sv.fqdn = rs.Fqdn
		sv.protocol = rs.Protocol
		sv.lPort = rs.LPort
		sv.tPort = rs.TPort

		SvcRoute.m[key] = sv

		//lprintf(4, "[INFO] get shere svc data(%s) \n", sv.tServer)

		// router lsa
		rlsa, exists := LSDB.m[cls.ListenIP]
		if !exists {
			rlsa = make(map[string]int)
		}

		_, exists = rlsa[sv.tServer]
		if !exists {
			rlsa[sv.tServer] = 99999999
		}

		// svc lsa
		lsa, exists := LSDB.m[sv.tServer]
		if !exists {
			lsa = make(map[string]int)
		}

		for j := 0; j < len(rs.NodeList); j++ {

			_, exists = lsa[rs.NodeList[j]]
			if !exists {
				lsa[rs.NodeList[j]] = 99999999 // 초기값
			}

			// router lsa
			_, exists = rlsa[rs.NodeList[j]]
			if !exists && rs.NodeList[j] != cls.ListenIP {
				rlsa[rs.NodeList[j]] = 99999999
			}
		}

		LSDB.m[sv.tServer] = lsa
		LSDB.m[cls.ListenIP] = rlsa
	}

	/*
		for key, val := range SvcRoute.m {
			lprintf(4, "[INFO] svc data key(%s), server ip(%s) \n", key, val.tServer)
		}

	// test router ip 192.168.41.99 ~ 192.168.41.108
	// test service ip 192.168.41.49:80, 192.168.41.75:80

	return SUCCESS
}
*/
func printRespMsg(r ResponseMsg) {

	lprintf(4, "[INFO] resp router id(%d) \n", r.RouterID)
	lprintf(4, "[INFO] resp router ver(%s) \n", r.Version)
	lprintf(4, "[INFO] resp router ip(%s) \n", r.NodeIP)

	lprintf(4, "------------------------------------------------")

	for i := 0; i < len(r.ServiceList); i++ {

		lprintf(4, "[INFO] service fqdn(%s) \n", r.ServiceList[i].Fqdn)
		lprintf(4, "[INFO] service ip(%s) \n", r.ServiceList[i].ServiceIP)
		lprintf(4, "[INFO] service port(%s) \n", r.ServiceList[i].TPort)
		lprintf(4, "[INFO] service listen port(%s) \n", r.ServiceList[i].LPort)
		lprintf(4, "[INFO] service protocol(%d) 0-HTTP, 1-TCP, 2-UDP \n", r.ServiceList[i].Protocol)

		for j := 0; j < len(r.ServiceList[i].NodeList); j++ {
			lprintf(4, "[INFO] node ip(%s) \n", r.ServiceList[i].NodeList[j])
		}

		lprintf(4, "------------------------------------------------")
	}
}

func notifyRead(fr, fe string, t int) {

	lprintf(4, "[INFO] notifyRead (%d)sec, files(%s,%s)\n", t, fe, fr)
	d := time.Duration(t)

	for {
		time.Sleep(d * time.Second)
		if _, err := os.Stat(fe); os.IsNotExist(err) {
			continue
		}

		v, r := cls.GetTokenValue("SPHERE", fr)
		if r != cls.CONF_ERR {
			lprintf(4, "[INFO] get sphere ip(%s)", v)
			cls.SetServerIp(cls.TCP_SPHERE, v)
		}

		os.Remove(fe)
		os.Remove(fr)
	}
}

func getPublic(port string) string {

	addrs := fmt.Sprintf("http://%s:%s/smartagent/getPublic", cls.ListenIP, port)
	req, err := http.NewRequest("GET", addrs, nil)
	if err != nil {
		lprintf(1, "[ERROR] http new request err(%s) \n", err.Error())
		return ""
	}

	//req.Header.Add("User-Agent", "Crawler")
	req.Header.Set("Connection", "close")
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	return string(data)
}
