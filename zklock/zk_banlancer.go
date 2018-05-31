package zklock

import (
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/Loopring/relay-lib/utils"
	"github.com/Loopring/relay/log"
	"encoding/json"
	"sync"
	"time"
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
	Name       string
	WorkerPath string
	EventPath  string
	Tasks      map[string]Task
	IsMaster   bool
	Mutex      sync.Mutex
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
	if zb.WorkerPath, err = zb.createSubPath(workerPath); err != nil {
		return err
	}
	if zb.EventPath, err = zb.createSubPath(eventPath); err != nil {
		return err
	}
	return nil
}

func (zb *ZkBalancer) Start() {
	zb.startMaster()
	zb.registerWorker()
}

func (zb *ZkBalancer) Stop() {
	if zb.IsMaster {
		ReleaseLock(zb.masterLockName())
	}
	zb.unRegisterWorker()
}

//slave callback
func (zb *ZkBalancer) OnAssign(assignFunc func(tasks []Task) error) {
	assignFunc([]Task{})
}

func (zb *ZkBalancer) Released(tasks []Task) error {
	if len(tasks) == 0 {
		log.Errorf("released tasks is empty")
		return fmt.Errorf("released tasks is empty")
	}
	if data, err := json.Marshal(tasks); err != nil {
		return fmt.Errorf("marshal released tasks failed %s", err.Error())
	} else {
		if _, err := ZkClient.CreateProtectedEphemeralSequential(fmt.Sprintf("%s/released-", zb.EventPath), data, zk.WorldACL(zk.PermAll)); err != nil {
			log.Errorf("released event node failed create", err.Error())
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
			if children, _, ch, err := ZkClient.ChildrenW(zb.WorkerPath); err != nil {
				if err != nil {
					log.Errorf("master get workers children failed %s\n", err.Error())
					ReleaseLock(zb.masterLockName())
					continue
				}
				if err := zb.loadTasks(children); err != nil {
					log.Errorf("master failed load tasks with error %s", err.Error())
					ReleaseLock(zb.masterLockName())
					continue
				}
				if err := zb.deprecateTasks(); err != nil {
					log.Errorf("master failed deprecateTasks with error %s", err.Error())
					ReleaseLock(zb.masterLockName())
					continue
				}
				go func() {
					for {
						select {
						case evt := <-ch:
							if evt.Type == zk.EventNodeChildrenChanged {
								if children, _, chx, err := ZkClient.ChildrenW(zb.WorkerPath); err == nil {
									ch = chx
									balanceTasks(children)
								}
							}
						}
					}
				}()
			}
		}
	}()
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

func (zb *ZkBalancer) assignedTasks(child string) error {
	assignedTasks := make([]Task, 3)
	for _, task := range zb.Tasks {
		if task.Owner == child && task.Status == Assigned {
			assignedTasks = append(assignedTasks, task)
		}
	}
	if data, err := json.Marshal(assignedTasks); err != nil {
		log.Errorf("assignedTasks encode worker %s failed, with error %s\n", child, err.Error())
		return err
	} else {
		if _, err := ZkClient.Set(fmt.Sprintf("%s/%s", zb.WorkerPath, child), data,-1); err != nil {
			log.Errorf("assign tasks for worker %s failed, with error %s\n", child, err.Error())
			return err
		} else {
			return nil
		}
	}
}

func (zb *ZkBalancer) loadTasks(children []string) error {
		for _, child := range children {
			if data, _, err := ZkClient.Get(fmt.Sprintf("%$s/%s", zb.WorkerPath, child)); err != nil {
				log.Errorf("Failed get worker data for %s\n, with error : %s", child, err.Error())
				return err
			} else {
				if workerTasks, err := decodeData(data); err != nil {
					log.Errorf("Failed decode tasks from worker %s, with error %s", child, err.Error())
					return err
				} else {
					for _, wt := range workerTasks {
						if v, ok := zb.Tasks[wt.Path]; ok {
							v.Owner = child
							v.Status = Assigned
						} else {
							wt.Status = Deleting
							wt.Timestamp = time.Now().Unix()
							zb.Tasks[wt.Path] = wt
						}
					}
				}
			}
		}
	return nil
}

func (zb *ZkBalancer) registerWorker() error {
	if _, err := ZkClient.Create(zb.workerEphemeral(), nil, zk.FlagEphemeral, zk.WorldACL(zk.PermAll)); err != nil {
		return nil
	} else {
		return nil
	}
}

func (zb *ZkBalancer) handleReleasedEvents() {
	_, _, ch, err := ZkClient.ChildrenW(zb.EventPath)
	if err != nil {
		log.Error("Watch event children failed")
	} else {
		go func() {
			for {
				select {
				case evt := <-ch:
					if evt.Type == zk.EventNodeChildrenChanged {
						if children, _, chx, err := ZkClient.ChildrenW(zb.EventPath); err == nil {
							ch = chx
							if len(children) > 0 {
								releaseMap := make(map[string]Task)
								for _, child := range children {
									if data, _, err := ZkClient.Get(fmt.Sprintf("%s/%s", zb.EventPath, child)); err != nil {
										log.Error("failed get event data from zk")
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
	if err := ZkClient.Delete(zb.workerEphemeral(), -1); err == nil {
		log.Error("failed unregister ")
		return err
	}
	return nil
}

func (zb *ZkBalancer) innerOnReleased(releasedTasks map[string]Task) {
	zb.Mutex.Lock()
	for _, task, := range zb.Tasks {
		switch task.Status {
		case Init:
			log.Errorf("Try release task with status Init : %+v\n", task)
			break
		case Assigned:
			log.Errorf("Try release task with status Assigned : %+v\n", task)
			break
		case Deleting:
			delete(zb.Tasks, task.Path)
			break
		case Releasing:
			task.Status = Init
			break
		}
	}
	zb.Mutex.Unlock()
	initTasks := make([]Task, len(zb.Tasks))
	for _, task, := range zb.Tasks {
		if task.Status == Init {
			initTasks = append(initTasks, task)
		}
	}
	if len(initTasks) > 0 {
		if children, _, err := ZkClient.Children(zb.WorkerPath); err != nil {
			log.Errorf("innerOnReleased failed get workers from zk %s\n", err.Error())
		} else {
			balanceTasks(children)
		}
	}
}

func balanceTasks(children []string) {
	
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

func (zb *ZkBalancer) workerEphemeral() string {
	return fmt.Sprintf("%s/%s", zb.WorkerPath, utils.GetLocalIp())
}

func decodeData(data []byte) ([]Task, error){
	releasedTasks := []Task{}
	if err := json.Unmarshal(data, &releasedTasks); err != nil {
		return nil, err
	} else {
		return releasedTasks, nil
	}
}
