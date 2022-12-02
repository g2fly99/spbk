package sip_server

import (
	"github.com/1lann/go-sip/sipnet"
	"github.com/beego/beego/v2/core/logs"
	"spbk/black"
	"spbk/statics"
)

func busy(r *sipnet.Request, conn *sipnet.Conn) error {
	resp := sipnet.NewResponse()
	resp.StatusCode = sipnet.StatusBusyHere
	return resp.SendTo(conn, r)
}

func blackHere(r *sipnet.Request, conn *sipnet.Conn) error {
	resp := sipnet.NewResponse()
	resp.StatusCode = sipnet.StatusBusyHere
	//resp.StatusCode = sipnet.StatusBusyHere
	return resp.SendTo(conn, r)
}

func refuse(r *sipnet.Request, conn *sipnet.Conn) error {
	resp := sipnet.NewResponse()
	resp.StatusCode = sipnet.StatusNoResponse

	return resp.SendTo(conn, r)
}

func handleInvite(r *sipnet.Request, conn *sipnet.Conn) {

	_, to, err := sipnet.ParseUserHeader(r.Header)
	if err != nil {
		logs.Debug("sip 请求解析错误:%v,r:%v", err, r)
		resp := sipnet.NewResponse()
		resp.BadRequest(conn, r, "Failed to parse From or To header.")
		return
	}

	telephone := to.URI.Username
	logs.Debug("[%v]receive invte", telephone)
	if len(telephone) == 0 || len(telephone) != 11 {
		logs.Error("被叫号码长度不对:[%v]", telephone)
		refuse(r, conn)
		return
	}

	statics.AddVoiceRqStatics(1)
	isBlack, err := black.VerifyWithTypeByIndex(telephone, "voice")
	if err != nil {
		err2 := blackHere(r, conn)
		logs.Debug("send busy:%v,err:%v,发送结果:%v", telephone, err, err2)
		return
	}

	if isBlack {
		err := blackHere(r, conn)
		logs.Debug("[%v]send busy to [%v],result:%v", telephone, conn.Address.String(), err)
	} else {
		refuse(r, conn)
		logs.Debug("[%v]send refuse to [%v]", telephone, conn.Address.String())
	}
}

func Init() {

	listener, err := sipnet.Listen("0.0.0.0:5072")
	if err != nil {
		panic(err)
	}
	go func() {
		logs.Debug("初始化sip服务器....")
		defer listener.Close()

		for {
			req, conn, err := listener.AcceptRequest()
			if err != nil {
				logs.Error("accept request error:", err)
				continue
			}

			//logs.Debug("received request,method:[%v]:%v", req.Method, req.Header)
			switch req.Method {
			case sipnet.MethodInvite:

				go handleInvite(req, conn)
			case sipnet.MethodAck:
			default:
				logs.Error("unknown method:%v,drop it", req.Method)
			}
		}
	}()
}
