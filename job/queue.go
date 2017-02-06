// Copyright (C) 2016  Juniper Networks, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package job

import (
	"sync"

	l "github.com/cloudwan/gohan/log"
)

//Job represents job
type Job struct {
	task func()
}

//Run executes one Job
func (job *Job) Run() {
	defer l.LogPanic(log)
	job.task()
}

//NewJob creates new Job
func NewJob(task func()) Job {
	return Job{
		task: task,
	}
}

//Worker subcribes manager's work queue and execute it
type Worker struct {
	queue   *Queue
	CloseCh chan bool
}

//NewWorker makes new working process
func NewWorker(queue *Queue) *Worker {
	return &Worker{
		queue:   queue,
		CloseCh: make(chan bool),
	}
}

//Start consuming task
func (worker *Worker) Start() {
	go func() {
		l.LogPanic(log)
		for {
			select {
			case job := <-worker.queue.queue:
				job.Run()
				worker.queue.waitGroup.Done()
			case <-worker.CloseCh:
				return
			}
		}
	}()
}

//Queue manages workers and job
type Queue struct {
	queue     chan Job
	workers   []*Worker
	waitGroup sync.WaitGroup
}

//NewQueue makes new manager
func NewQueue(maxWorker uint) *Queue {
	queue := &Queue{
		queue:   make(chan Job),
		workers: make([]*Worker, maxWorker),
	}
	for i := uint(0); i < maxWorker; i++ {
		worker := NewWorker(queue)
		worker.Start()
		queue.workers[i] = worker
	}
	return queue
}

//Add new job
func (queue *Queue) Add(job Job) {
	queue.waitGroup.Add(1)
	queue.queue <- job
}

//Wait until all job get done
func (queue *Queue) Wait() {
	queue.waitGroup.Wait()
}

//Stop stops all worker after all job get done
func (queue *Queue) Stop() {
	queue.waitGroup.Wait()
	queue.ForceStop()
}

//ForceStop forces job manager without waiting job complete
func (queue *Queue) ForceStop() {
	for _, worker := range queue.workers {
		worker.CloseCh <- true
	}
}
