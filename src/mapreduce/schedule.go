package mapreduce

import (
	"sync"
	"fmt"
)
//
// schedule() starts and waits for all tasks in the given phase (Map
// or Reduce). the mapFiles argument holds the names of the files that
// are the inputs to the map phase, one per map task. nReduce is the
// number of reduce tasks. the registerChan argument yields a stream
// of registered workers; each item is the worker's RPC address,
// suitable for passing to call(). registerChan will yield all
// existing registered workers (if any) and new ones as they register.
//
func schedule(jobName string, mapFiles []string, nReduce int, phase jobPhase, registerChan chan string) {
	var ntasks int
	var n_other int // number of inputs (for reduce) or outputs (for map)
	switch phase {
	case mapPhase:
		ntasks = len(mapFiles)
		n_other = nReduce
	case reducePhase:
		ntasks = nReduce
		n_other = len(mapFiles)
	}

	fmt.Printf("Schedule: %v %v tasks (%d I/Os)\n", ntasks, phase, n_other)

	// All ntasks tasks have to be scheduled on workers, and only once all of
	// them have been completed successfully should the function return.
	// Remember that workers may fail, and that any given worker may finish
	// multiple tasks.
	//
	// TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO
	//
	var wg sync.WaitGroup
	for i := 0; i < ntasks; i++ {
		var args DoTaskArgs
		if phase == mapPhase {
			args = DoTaskArgs{JobName: jobName, File: mapFiles[i], Phase: phase, TaskNumber: i, NumOtherPhase: n_other}
		} else {
			args = DoTaskArgs{JobName: jobName, Phase: phase, TaskNumber: i, NumOtherPhase: n_other}
		}
		wg.Add(1)
		go func(args DoTaskArgs){
			retry := 0
			for {
				nowWorker := <- registerChan
				// fmt.Printf("Assign to %d: task %d, phase %v, retry %d\n", nowWorker, args.TaskNumber, args.Phase, retry)
				retry += 1 // For debug purpose only
				ok := call(nowWorker, "Worker.DoTask", args, nil)
				if ok {
					wg.Done()
					registerChan <- nowWorker
					break
				}
			}
		}(args)
	}
	wg.Wait()
	fmt.Printf("Schedule: %v phase done\n", phase)
	return
}
