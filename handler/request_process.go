package handler

import (
	"encoding/json"
	"github.com/valyala/fasthttp"
	"log"
	"net/http"
	"sync"

	"github.com/gongshen/xxl-job-client/admin"
	"github.com/gongshen/xxl-job-client/transport"
)

type RequestProcess struct {
	sync.RWMutex

	adminServer *admin.XxlAdminServer

	JobHandler *JobHandler

	ReqHandler *HttpRequestHandler
}

func NewRequestProcess(adminServer *admin.XxlAdminServer, handler *HttpRequestHandler) *RequestProcess {
	requestHandler := &RequestProcess{
		adminServer: adminServer,
		ReqHandler:  handler,
	}
	jobHandler := &JobHandler{
		QueueMap:     make(map[int32]*JobQueue),
		CallbackFunc: requestHandler.jobRunCallback,
	}
	requestHandler.JobHandler = jobHandler
	return requestHandler
}

func (r *RequestProcess) RegisterJob(jobName string, function JobHandlerFunc) {
	r.JobHandler.RegisterJob(jobName, function)
}

func (r *RequestProcess) pushJob(trigger *transport.TriggerParam) {
	err := r.JobHandler.PutJobToQueue(trigger)
	if err != nil {
		log.Printf("PutJobToQueue err. jobId:%d,err:%v\n", trigger.JobId, err)

		callback := &transport.HandleCallbackParam{
			LogId:      trigger.LogId,
			LogDateTim: trigger.LogDateTime,
			Code:       http.StatusInternalServerError,
			Msg:        err.Error(),
		}
		if ne, ok := err.(interface{ Temporary() bool }); ok && ne.Temporary() {
			callback.Code = http.StatusOK
		}

		r.adminServer.CallbackAdmin([]*transport.HandleCallbackParam{callback})
	}
}

func (r *RequestProcess) jobRunCallback(logId, logDatetime int64, runErr error) {
	callback := &transport.HandleCallbackParam{
		LogId:      logId,
		LogDateTim: logDatetime,
		Code:       http.StatusOK,
		Msg:        "success",
	}
	if runErr != nil {
		if ne, ok := runErr.(interface{ Temporary() bool }); !ok || !ne.Temporary() {
			callback.Code = http.StatusInternalServerError
			callback.Msg = runErr.Error()
		} else {
			callback.Msg = runErr.Error()
		}
	}
	r.adminServer.CallbackAdmin([]*transport.HandleCallbackParam{callback})
}

func (r *RequestProcess) RequestProcess(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Request.URI().Path())
	returnt := transport.ReturnT{
		Code: http.StatusOK,
		Msg:  "success",
	}
	switch path {
	case "/idleBeat":
		jobId, err := r.ReqHandler.IdleBeat(ctx.Request.Body())
		if err != nil {
			returnt.Code = http.StatusInternalServerError
			returnt.Msg = err.Error()
		} else {
			if r.JobHandler.HasRunning(jobId) {
				returnt.Code = http.StatusInternalServerError
				returnt.Msg = "the server busy"
			}
		}
	case "/log":
		l, err := r.ReqHandler.Log(ctx.Request.Body())
		if err != nil {
			returnt.Code = http.StatusInternalServerError
			returnt.Msg = err.Error()
		} else {
			returnt.Content = l
		}
	case "/kill":
		jobId, err := r.ReqHandler.Kill(ctx.Request.Body())
		if err != nil {
			returnt.Code = http.StatusInternalServerError
			returnt.Msg = err.Error()
		} else {
			r.JobHandler.cancelJob(jobId)
		}
	default:
		ta, err := r.ReqHandler.Run(ctx.Request.Body())
		if err != nil {
			log.Printf("PushJob. triggerParams: %+v\n", ta)
			returnt.Code = http.StatusInternalServerError
			returnt.Msg = err.Error()
		}
		go r.pushJob(ta)
	}

	bytes, _ := json.Marshal(&returnt)
	ctx.Success("application/json", bytes)
	return
}

func (r *RequestProcess) RemoveRegisterExecutor() {
	r.JobHandler.clearJob()
	r.adminServer.RemoveRegisterExecutor()
}

func (r *RequestProcess) RegisterExecutor() {
	r.adminServer.RegisterExecutor()
	go r.adminServer.AutoRegisterJobGroup()
}
