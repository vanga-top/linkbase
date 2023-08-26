package task

type taskQueue interface {
	utChan() <-chan int
	utEmpty() bool
	utFull() bool
	addUnissuedTask(t task)
	FrontUnissuedTask() task
	PopUnissuedTask() task
	AddActiveTask(t task)
	PopActiveTask(taskID UniqueID) task
	getTaskByReqID(reqID UniqueID) task
	Enqueue(t task) error
	setMaxTaskNum(num int64)
	getMaxTaskNum() int64
}
