package zklock

import (
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/Loopring/relay-lib/utils"
	"github.com/Loopring/relay/log"
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

type ZkBalancer struct {
	Name           string
	WorkerBasePath string
	EventPath      string
	Tasks          map[string]Task
	IsMaster       bool
	Mutex          sync.Mutex
	OnAssignFunc   func([]Task) error
}

type Status int

const (
	Init      = iota
	Assigned
	Releasing
	Deleting
)

func (zb *ZkBalancer) Init(tasks []Task) error {
	if zb.Name == "" {
		return fmt.Errorf("balancer Name is empty")
	}
	if len(zb.Tasks) == 0 {
		zb.Tasks = make(map[string]Task)
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
			zb.Tasks[task.Path] = task
		}
		return fmt.Errorf("no tasks to balance")
	}
	if !IsInit() {
		return fmt.Errorf("zkClient is not intiliazed")
	}
	var err error
	if zb.WorkerBasePath, err = zb.createSubPath(workerPath); err != nil {
		return err
	}
	if zb.EventPath, err = zb.createSubPath(eventPath); err != nil {
		return err
	}
	return nil
}

func (zb *ZkBalancer) Start() {
	zb.startMaster()
	time.Sleep(time.Second * time.Duration(2))
	zb.registerWorker()
}

func (zb *ZkBalancer) Stop() {
	if zb.IsMaster {
		ReleaseLock(zb.masterLockName())
	}
	zb.unRegisterWorker()
}

//worker callback
func (zb *ZkBalancer) OnAssign(assignFunc func(tasks []Task) error) {
	zb.OnAssignFunc = assignFunc
}

func (zb *ZkBalancer) Released(tasks []Task) error {
	if len(tasks) == 0 {
		log.Errorf("released tasks is empty\n")
		return fmt.Errorf("released tasks is empty")
	}
	if data, err := json.Marshal(tasks); err != nil {
		return fmt.Errorf("marshal released tasks failed %s", err.Error())
	} else {
		if _, err := ZkClient.CreateProtectedEphemeralSequential(fmt.Sprintf("%s/released-", zb.EventPath), data, zk.WorldACL(zk.PermAll)); err != nil {
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
			zb.IsMaster = false
			if !firstTime {
				time.Sleep(time.Second * time.Duration(3))
			} else {
				firstTime = false
			}
			if err := TryLock(zb.masterLockName()); err != nil {
				log.Errorf("master failed get lock for %s, with error %s, try again\n", zb.masterLockName(), err.Error())
				continue
			}
			zb.IsMaster = true
			if workers, _, ch, err := ZkClient.ChildrenW(zb.WorkerBasePath); err != nil {
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
								if children, _, chx, err := ZkClient.ChildrenW(zb.WorkerBasePath); err == nil {
									ch = chx
									zb.balanceTasks(children)
								}
							}
						}
					}
				}()
				break
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
	if data, _, err := ZkClient.Get(fmt.Sprintf("%s/%s", zb.WorkerBasePath, worker)); err != nil {
		log.Errorf("loadTasksForWorker failed on get worker data for %s\n, with error : %s\n", worker, err.Error())
		return err
	} else {
		if workerTasks, err := decodeData(data); err != nil {
			log.Errorf("loadTasksForWorker failed on decode tasks from worker %s, with error %s\n", worker, err.Error())
			return err
		} else {
			for _, wt := range workerTasks {
				if v, ok := zb.Tasks[wt.Path]; ok {
					v.Owner = worker
					v.Status = Assigned
					v.Timestamp = time.Now().Unix()
				} else {
					wt.Status = Deleting
					wt.Timestamp = time.Now().Unix()
					zb.Tasks[wt.Path] = wt
				}
			}
		}
	}
	return nil
}

func (zb *ZkBalancer) deprecateTasks() error {
	needDeleteWorker := make(map[string]string)
	var err error = nil
	for _, task := range zb.Tasks {
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
	assignedTasks := make([]Task, 10)
	for _, task := range zb.Tasks {
		if task.Owner == worker && task.Status == Assigned {
			assignedTasks = append(assignedTasks, task)
		}
	}
	if data, err := json.Marshal(assignedTasks); err != nil {
		log.Errorf("assignedTasks encode worker %s data failed, with error %s\n", worker, err.Error())
		return err
	} else {
		if _, err := ZkClient.Set(fmt.Sprintf("%s/%s", zb.WorkerBasePath, worker), data,-1); err != nil {
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
				if err := zb.loadTasksForWorker(utils.GetLocalIp()); err != nil {
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
										zb.OnAssignFunc(workerTasks)
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
	_, _, ch, err := ZkClient.ChildrenW(zb.EventPath)
	if err != nil {
		log.Errorf("Watch event children failed with error : %s\n", err.Error())
	} else {
		go func() {
			for {
				select {
				case evt := <-ch:
					if evt.Type == zk.EventNodeChildrenChanged {
						if events, _, chx, err := ZkClient.ChildrenW(zb.EventPath); err == nil {
							ch = chx
							if len(events) > 0 {
								releaseMap := make(map[string]Task)
								for _, event := range events {
									if data, _, err := ZkClient.Get(fmt.Sprintf("%s/%s", zb.EventPath, event)); err != nil {
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
	zb.Mutex.Lock()
	hasInitTasks := false
	for _, rlt, := range releasedTasks {
		if origin, ok := zb.Tasks[rlt.Path]; ok {
			switch origin.Status {
			case Init:
				log.Errorf("Try release task with status Init : %+v\n", origin)
				break
			case Assigned:
				log.Errorf("Try release task with status Assigned : %+v\n", origin)
				break
			case Deleting:
				delete(zb.Tasks, origin.Path)
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
	zb.Mutex.Unlock()
	if hasInitTasks {
		if workers, _, err := ZkClient.Children(zb.WorkerBasePath); err != nil {
			log.Errorf("innerOnReleased failed get workers from zk %s\n", err.Error())
		} else {
			zb.balanceTasks(workers)
		}
	}
}

func (zb *ZkBalancer) balanceTasks(workers []string) {
	sortedTask := make([]Task, len(zb.Tasks))
	for _, t := range zb.Tasks {
		if t.Status == Assigned || t.Status == Init {
			sortedTask = append(sortedTask, t)
		}
	}
	if len(sortedTask) == 0 {
		return
	}
	sort.Sort(tasksForSort(sortedTask))
	workersWeight := make(map[string]int)
	for i := 0; i < len(workers) && i < len(sortedTask); i++ {
		workersWeight[workers[i]] = sortedTask[i].Weight
		sortedTask[i].Owner = workers[i]
		sortedTask[i].Status = Assigned
		sortedTask[i].Timestamp = time.Now().Unix()
	}
	if len(sortedTask) > len(workers) {
		for i := len(workers); i < len(sortedTask); i++ {
			minWorker := minWeightWorker(workersWeight)
			sortedTask[i].Owner = minWorker
			sortedTask[i].Status = Assigned
			sortedTask[i].Timestamp = time.Now().Unix()
			workersWeight[minWorker] += workersWeight[minWorker] + sortedTask[i].Weight
		}
	}
	for _, worker := range workers {
		zb.assignedTasks(worker)
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

func (zb *ZkBalancer) createSubPath(subPath string) (string, error) {
	subPathFull := fmt.Sprintf("%s/%s/%s", balancerShareBasePath, zb.Name, subPath)
	isExist, _, err := ZkClient.Exists(subPathFull)
	if err != nil {
		return "", fmt.Errorf("check zk path %s exists failed : %s", subPathFull, err.Error())
	}
	if !isExist {
		_, err := ZkClient.Create(subPathFull, nil, 0, zk.WorldACL(zk.PermAll))
		if err != nil && err != zk.ErrNodeExists {
			return "", fmt.Errorf("failed create balancer sub path %s", subPathFull)
		}
	}
	return subPathFull, nil
}

func (zb *ZkBalancer) masterLockName() string {
	return fmt.Sprintf("balancer/%s", zb.Name)
}

func (zb *ZkBalancer) workerEphemeralPath() string {
	return fmt.Sprintf("%s/%s", zb.WorkerBasePath, utils.GetLocalIp())
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
