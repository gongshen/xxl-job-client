package handler

import (
	"context"
	"errors"
	"github.com/gongshen/xxl-job-client/constants"
	"log"
	"sync"
	"sync/atomic"

	"github.com/gongshen/xxl-job-client/logger"
	"github.com/gongshen/xxl-job-client/queue"
	"github.com/gongshen/xxl-job-client/transport"
)

type JobHandlerFunc func(ctx context.Context) error

type JobQueue struct {
	JobId    int32
	GlueType string
	ExecuteHandler
	CurrentJob *JobRunParam
	Run        int32 //0 stop, 1 run
	Queue      *queue.Queue
	Callback   func(int64, int64, error)
}

type JobRunParam struct {
	LogId                 int64
	LogDateTime           int64
	JobName               string
	JobTag                string
	InputParam            map[string]interface{}
	ShardIdx              int32
	ShardTotal            int32
	CurrentCancelFunc     context.CancelFunc
	ExecutorBlockStrategy string
}

func (jq *JobQueue) StartJob() {
	if atomic.CompareAndSwapInt32(&jq.Run, 0, 1) {
		jq.asyRunJob()
	}
}

func (jq *JobQueue) StopJob() bool {
	return atomic.CompareAndSwapInt32(&jq.Run, 1, 0)
}

func (jq *JobQueue) asyRunJob() {
	go func() {
		for {
			has, node := jq.Queue.Poll()
			if has {
				jq.CurrentJob = node.(*JobRunParam)
				jq.Callback(jq.CurrentJob.LogId, jq.CurrentJob.LogDateTime, jq.Execute(jq.JobId, jq.GlueType, jq.CurrentJob))
			} else {
				jq.StopJob()
				break
			}
		}
	}()
}

type JobHandler struct {
	sync.RWMutex

	jobMap map[string]JobHandlerFunc

	QueueMap map[int32]*JobQueue

	CallbackFunc func(int64, int64, error)
}

func (j *JobHandler) BeanJobLength() int {
	if j.jobMap == nil {
		return 0
	}
	return len(j.jobMap)
}

func (j *JobHandler) RegisterJob(jobName string, function JobHandlerFunc) {
	j.Lock()
	defer j.Unlock()
	if j.jobMap == nil {
		j.jobMap = make(map[string]JobHandlerFunc)
	} else {
		_, ok := j.jobMap[jobName]
		if ok {
			panic("the job had already register, job name can't be repeated:" + jobName)
		}
	}
	j.jobMap[jobName] = function
}

func (j *JobHandler) HasRunning(jobId int32) bool {
	qu, has := j.QueueMap[jobId]
	if has {
		if qu.Run > 0 || qu.Queue.HasNext() {
			return true
		}
	}
	return false
}

func (j *JobHandler) PutJobToQueue(trigger *transport.TriggerParam) (err error) {
	qu, has := j.QueueMap[trigger.JobId] //map value是地址，读不加锁
	if has {
		if trigger.ExecutorBlockStrategy == constants.DiscardLater {
			if atomic.LoadInt32(&qu.Run) == 1 {
				// 丢弃本次调度
				return &ExecutorBlockStrategyErr{
					msg: "job正在执行，丢弃本地调度",
				}
			}
		} else if trigger.ExecutorBlockStrategy == constants.CoverEarly {
			if atomic.LoadInt32(&qu.Run) == 1 {
				// 杀掉队列中的任务
				j.cancelJob(trigger.JobId)
				goto initQueue
			}
		}
		runParam, err := qu.ParseJob(trigger)
		if err != nil {
			return err
		}
		err = qu.Queue.Put(runParam)
		if err == nil {
			qu.StartJob()
			return err
		}
		return err
	}

initQueue:
	j.Lock() //任务map初始化锁
	defer j.Unlock()

	jobQueue := &JobQueue{
		GlueType: trigger.GlueType,
		JobId:    trigger.JobId,
		Callback: j.CallbackFunc,
	}
	if trigger.ExecutorHandler != "" {
		if j.jobMap == nil && len(j.jobMap) <= 0 {
			return errors.New("bean job handler not found")
		}
		fun, ok := j.jobMap[trigger.ExecutorHandler]
		if !ok {
			return errors.New("bean job handler not found")
		}

		jobQueue.ExecuteHandler = &BeanHandler{
			RunFunc: fun,
		}
	} else {
		jobQueue.ExecuteHandler = &ScriptHandler{}
	}

	runParam, err := jobQueue.ParseJob(trigger)
	if err != nil {
		return err
	}
	q := queue.NewQueue()
	err = q.Put(runParam)
	if err != nil {
		return err
	}

	jobQueue.Queue = q
	j.QueueMap[trigger.JobId] = jobQueue
	jobQueue.StartJob()
	return err
}

func (j *JobHandler) cancelJob(jobId int32) {
	queue, has := j.QueueMap[jobId]
	if has {
		log.Print("job be canceled, id:", jobId)
		res := queue.StopJob()
		if res {
			if queue.CurrentJob != nil && queue.CurrentJob.CurrentCancelFunc != nil {
				queue.CurrentJob.CurrentCancelFunc()

				go func() {
					jobParam := make(map[string]map[string]interface{})
					logParam := make(map[string]interface{})
					logParam["logId"] = queue.CurrentJob.LogId
					logParam["jobId"] = jobId
					logParam["jobName"] = queue.CurrentJob.JobName
					logParam["jobFunc"] = queue.CurrentJob.JobTag
					jobParam["logParam"] = logParam
					ctx := context.WithValue(context.Background(), "jobParam", jobParam)
					logger.Info(ctx, "job canceled by admin!")
				}()
			}
			if queue.Queue != nil {
				queue.Queue.Clear()
			}
		}
	}
}

func (j *JobHandler) clearJob() {
	j.jobMap = map[string]JobHandlerFunc{}
	j.QueueMap = make(map[int32]*JobQueue)
}
