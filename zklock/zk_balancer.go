package zklock

import (
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/Loopring/relay-lib/utils"
	"github.com/Loopring/relay-lib/log"
	"encoding/json"
	"sync"
	"time"
	"sort"
)

type Task struct {
	Payload string //Native info for bussiness structs
	Path    string // friendly string for zk path composite, default is .substr(Payload, 0, 10)
	Weight  int    //bi
	Status  int
	Owner string
	Timestamp int64
}

const balancerShareBasePath = "/loopring_balancer"
const workerPath = "worker"
const eventPath = "event"

const networkInterface = "eth0"

type ZkBalancer struct {
	name           string
	workerBasePath string
	eEventBasePath string
	tasks          map[string]Task
	isMaster       bool
	mutex          sync.Mutex
	onAssignFunc   func([]Task) error
	workerPath           string
}

type Status int

const (
	Init      = iota
	Assigned
	Releasing
	Deleting
)

func (zb *ZkBalancer) Init(name string, tasks []Task, path ...string) error {
	if name != ""{
		zb.name = name
	} else if zb.name == "" {
		return fmt.Errorf("balancer Name is empty")
	}
	if len(tasks) > 0 {
		zb.tasks = make(map[string]Task)
		for _, task := range tasks {
			if task.Payload == "" {
				return fmt.Errorf("task payload is empty")
			}
			if task.Path == "" {
				if len(task.Payload) <= 10 {
					task.Path = task.Payload
				} else {
					task.Path = task.Payload[:10]
				}
			}
			task.Status = Init
			zb.tasks[task.Path] = task
		}
	} else {
		return fmt.Errorf("no tasks to balance")
	}
	if !IsLockInitialed() {
		return fmt.Errorf("zkClient is not intiliazed")
	}
	var err error
	if _, err = zb.createPath(balancerShareBasePath); err != nil {
		return fmt.Errorf("create balancer base path failed %s", balancerShareBasePath)
	}
	blp := fmt.Sprintf("%s/%s", balancerShareBasePath, zb.name)
	if _, err = zb.createPath(blp); err != nil {
		return fmt.Errorf("create balancer path for %s failed with error : %s", blp , err.Error())
	}
	wp := fmt.Sprintf("%s/%s", blp, workerPath)
	if zb.workerBasePath, err = zb.createPath(wp); err != nil {
		return fmt.Errorf("create balancer worker path for %s failed with error : %s", wp , err.Error())
	}
	ep := fmt.Sprintf("%s/%s", blp, eventPath)
	if zb.eEventBasePath, err = zb.createPath(ep); err != nil {
		return fmt.Errorf("create balancer event path for %s failed with error : %s", ep , err.Error())
	}
	if len(path) > 0 {
		zb.workerPath = path[0]
	} else {
		zb.workerPath = utils.GetLocalIpByInterface(networkInterface)
	}
	zb.mutex = sync.Mutex{}
	return nil
}

func (zb *ZkBalancer) Start() {
	zb.startMaster()
	time.Sleep(time.Second * time.Duration(2))
	zb.registerWorker()
}

func (zb *ZkBalancer) Stop() {
	if zb.isMaster {
		ReleaseLock(zb.masterLockName())
	}
	zb.unRegisterWorker()
}

//worker callback
func (zb *ZkBalancer) OnAssign(assignFunc func(tasks []Task) error) {
	zb.onAssignFunc = assignFunc
}

func (zb *ZkBalancer) Released(tasks []Task) error {
	if len(tasks) == 0 {
		log.Errorf("released tasks is empty\n")
		return fmt.Errorf("released tasks is empty")
	}
	if data, err := json.Marshal(tasks); err != nil {
		return fmt.Errorf("marshal released tasks failed %s", err.Error())
	} else {
		if _, err := ZkClient.CreateProtectedEphemeralSequential(fmt.Sprintf("%s/released-", zb.eEventBasePath), data, zk.WorldACL(zk.PermAll)); err != nil {
			log.Errorf("released event node failed create with error : %s\n", err.Error())
			return err
		}
		return nil
	}
}

func (zb *ZkBalancer) startMaster() {
	go func() {
		firstTime := true
		for {
			zb.isMaster = false
			if !firstTime {
				time.Sleep(time.Second * time.Duration(3))
			} else {
				firstTime = false
			}
			if err := TryLock(zb.masterLockName()); err != nil {
				log.Errorf("master failed get lock for %s, with error %s, try again\n", zb.masterLockName(), err.Error())
				continue
			} else {
				log.Info("get master lock success")
			}
			zb.isMaster = true
			if workers, _, ch, err := ZkClient.ChildrenW(zb.workerBasePath); err == nil {
				if err != nil {
					log.Errorf("master get workers failed %s\n", err.Error())
					ReleaseLock(zb.masterLockName())
					continue
				}
				if err := zb.loadTasks(workers); err != nil {
					log.Errorf("master load tasks from worker failed with error %s\n", err.Error())
					ReleaseLock(zb.masterLockName())
					continue
				}
				if err := zb.deprecateTasks(); err != nil {
					log.Errorf("master failed deprecateTasks with error %s\n", err.Error())
					ReleaseLock(zb.masterLockName())
					continue
				}
				go func() {
					for {
						select {
						case evt := <-ch:
							if evt.Type == zk.EventNodeChildrenChanged {
								if children, _, chx, err := ZkClient.ChildrenW(zb.workerBasePath); err == nil {
									ch = chx
									zb.balanceTasks(children)
								}
							}
						}
					}
				}()
				return
			} else {
				log.Errorf("master watch workers failed with error : %s", err.Error())
			}
		}
	}()
}

func (zb *ZkBalancer) loadTasks(workers []string) error {
	for _, worker := range workers {
		if err := zb.loadTasksForWorker(worker); err != nil {
			return err
		}
	}
	return nil
}

func (zb *ZkBalancer) loadTasksForWorker(worker string) error {
	if data, _, err := ZkClient.Get(fmt.Sprintf("%s/%s", zb.workerBasePath, worker)); err != nil {
		log.Errorf("loadTasksForWorker failed on get worker data for %s\n, with error : %s\n", worker, err.Error())
		return err
	} else {
		if workerTasks, err := decodeData(data); err != nil {
			log.Errorf("loadTasksForWorker failed on decode tasks from worker %s, with error %s\n", worker, err.Error())
			return err
		} else {
			for _, wt := range workerTasks {
				if v, ok := zb.tasks[wt.Path]; ok {
					v.Owner = worker
					v.Status = Assigned
					v.Timestamp = time.Now().Unix()
				} else {
					wt.Status = Deleting
					wt.Timestamp = time.Now().Unix()
					zb.tasks[wt.Path] = wt
				}
			}
		}
	}
	return nil
}

func (zb *ZkBalancer) deprecateTasks() error {
	needDeleteWorker := make(map[string]string)
	var err error = nil
	for _, task := range zb.tasks {
		if task.Status == Deleting {
			if _, ok := needDeleteWorker[task.Owner]; !ok {
				needDeleteWorker[task.Owner] = "-"
				if e := zb.assignedTasks(task.Owner); e != nil {
					err = e
				}
			}
		}
	}
	return err
}

func (zb *ZkBalancer) assignedTasks(worker string) error {
	assignedTasks := make([]Task, 0, 10)
	for _, task := range zb.tasks {
		if task.Owner == worker && task.Status == Assigned {
			assignedTasks = append(assignedTasks, task)
		}
	}
	if data, err := json.Marshal(assignedTasks); err != nil {
		log.Errorf("assignedTasks encode worker %s data failed, with error %s\n", worker, err.Error())
		return err
	} else {
		if _, err := ZkClient.Set(fmt.Sprintf("%s/%s", zb.workerBasePath, worker), data,-1); err != nil {
			log.Errorf("assignedTasks  set data for worker %s failed, with error %s\n", worker, err.Error())
			return err
		} else {
			return nil
		}
	}
}

func (zb *ZkBalancer) registerWorker() error {
	for {
		loaded := false
		if _, err := ZkClient.Create(zb.workerEphemeralPath(), nil, zk.FlagEphemeral, zk.WorldACL(zk.PermAll)); err == nil || err == zk.ErrNodeExists {
			if err == zk.ErrNodeExists && !loaded {
				if err := zb.loadTasksForWorker(zb.workerPath); err != nil {
					loaded = true
				}
			}
			_, _, ch, err := ZkClient.GetW(zb.workerEphemeralPath())
			if err != nil {
				log.Errorf("watch worker %s failed with error %s\n", zb.workerEphemeralPath(),  err.Error())
				continue
			} else {
				go func() {
					for {
						select {
						case evt := <- ch:
							if evt.Type == zk.EventNodeDataChanged {
								if data, _, chx, err := ZkClient.GetW(zb.workerEphemeralPath()); err != nil {
									log.Errorf("Get worker %s data failed with error : %s\n", zb.workerEphemeralPath(), err.Error())
								} else {
									ch = chx
									if workerTasks, err := decodeData(data); err != nil {
										zb.onAssignFunc(workerTasks)
									}
								}
							}
						}
					}
				}()
			}
			return nil
		} else {
			log.Errorf("registerWorker failed when create zk node with error : %s\n", err.Error())
		}
	}
}

func (zb *ZkBalancer) handleReleasedEvents() {
	_, _, ch, err := ZkClient.ChildrenW(zb.eEventBasePath)
	if err != nil {
		log.Errorf("Watch event children failed with error : %s\n", err.Error())
	} else {
		go func() {
			for {
				select {
				case evt := <-ch:
					if evt.Type == zk.EventNodeChildrenChanged {
						if events, _, chx, err := ZkClient.ChildrenW(zb.eEventBasePath); err == nil {
							ch = chx
							if len(events) > 0 {
								releaseMap := make(map[string]Task)
								for _, event := range events {
									if data, _, err := ZkClient.Get(fmt.Sprintf("%s/%s", zb.eEventBasePath, event)); err != nil {
										log.Errorf("failed get event data from zk with error : %s\n", err.Error())
									} else {
										if releasedTasks, err:= decodeData(data); err == nil {
											for _, v := range releasedTasks {
												releaseMap[v.Path] = v
											}
										}
									}
								}
								zb.innerOnReleased(releaseMap)
							}
						}
					}
				}
			}
		}()
	}
}

func (zb *ZkBalancer) unRegisterWorker() error {
	if err := ZkClient.Delete(zb.workerEphemeralPath(), -1); err != nil {
		log.Errorf("failed unRegister worker with error : %s", err.Error())
		return err
	}
	return nil
}

func (zb *ZkBalancer) innerOnReleased(releasedTasks map[string]Task) {
	zb.mutex.Lock()
	hasInitTasks := false
	for _, rlt := range releasedTasks {
		if origin, ok := zb.tasks[rlt.Path]; ok {
			switch origin.Status {
			case Init:
				log.Errorf("Try release task with status Init : %+v\n", origin)
				break
			case Assigned:
				log.Errorf("Try release task with status Assigned : %+v\n", origin)
				break
			case Deleting:
				delete(zb.tasks, origin.Path)
				break
			case Releasing:
				origin.Status = Init
				hasInitTasks = true
				break
			}
		} else {
			log.Error("Try release task not exits, ignore\n")
		}
	}
	zb.mutex.Unlock()
	if hasInitTasks {
		if workers, _, err := ZkClient.Children(zb.workerBasePath); err != nil {
			log.Errorf("innerOnReleased failed get workers from zk %s\n", err.Error())
		} else {
			zb.balanceTasks(workers)
		}
	}
}

func (zb *ZkBalancer) balanceTasks(workers []string) {
	sortedTask := make([]Task, 0, len(zb.tasks))
	for _, t := range zb.tasks {
		if t.Status == Assigned || t.Status == Init {
			sortedTask = append(sortedTask, t)
		} else if t.Status == Releasing {
			log.Info("task with releasing status, not rebalance\n")
			return
		}
	}
	if len(sortedTask) == 0 {
		return
	}
	sort.Sort(tasksForSort(sortedTask))
	workersWeight := make(map[string]int)
	newReleasingCount := 0
	for i := 0; i < len(workers) && i < len(sortedTask); i++ {
		workersWeight[workers[i]] = sortedTask[i].Weight
		newReleasingCount += tryReAssignTask(sortedTask[i], workers[i])
	}
	if len(sortedTask) > len(workers) {
		for i := len(workers); i < len(sortedTask); i++ {
			minWorker := minWeightWorker(workersWeight)
			newReleasingCount += tryReAssignTask(sortedTask[i], minWorker)
			workersWeight[minWorker] += workersWeight[minWorker] + sortedTask[i].Weight
		}
	}
	if newReleasingCount == 0 {
		for _, worker := range workers {
			zb.assignedTasks(worker)
		}
	}
}

func tryReAssignTask(task Task, newWorker string) int {
	task.Timestamp = time.Now().Unix()
	if task.Status == Assigned {
		if task.Owner != newWorker {
			task.Status = Releasing
			return 1
		} else {
			return 0
		}
	} else {
		task.Status = Assigned
		task.Owner = newWorker
		return 0
	}
}

func minWeightWorker(workerWeight map[string]int) string {
	minWorker := ""
	minWeight := -1
	for worker, weight := range workerWeight {
		if minWeight == -1 || weight < minWeight {
			minWorker = worker
			minWeight = weight
		}
	}
	return minWorker
}

func (zb *ZkBalancer) createPath(path string) (string, error) {
	isExist, _, err := ZkClient.Exists(path)
	if err != nil {
		return "", fmt.Errorf("check zk path %s exists failed : %s", path, err.Error())
	}
	if !isExist {
		_, err := ZkClient.Create(path, nil, 0, zk.WorldACL(zk.PermAll))
		if err != nil && err != zk.ErrNodeExists {
			return "", fmt.Errorf("failed create balancer sub path %s ,with error : %s ", path, err.Error())
		}
	}
	return path, nil
}

func (zb *ZkBalancer) masterLockName() string {
	return fmt.Sprintf("balancer/%s", zb.name)
}

func (zb *ZkBalancer) workerEphemeralPath() string {
	return fmt.Sprintf("%s/%s", zb.workerBasePath, zb.workerPath)
}

func decodeData(data []byte) ([]Task, error) {
	releasedTasks := []Task{}
	if err := json.Unmarshal(data, &releasedTasks); err != nil {
		return nil, err
	} else {
		return releasedTasks, nil
	}
}

type tasksForSort []Task

func (a tasksForSort) Len() int           { return len(a) }
func (a tasksForSort) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a tasksForSort) Less(i, j int) bool {
	if a[i].Weight > a[j].Weight {
		return true
	} else {
		return false
	}
}