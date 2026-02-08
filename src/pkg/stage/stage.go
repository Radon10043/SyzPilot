package stage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Radon10043/cloud/src/pkg/agent"
	"github.com/Radon10043/cloud/src/pkg/pool"
	"github.com/Radon10043/cloud/src/pkg/queue"
)

// StageHelper is a helper struct to hold intermediate results during spec writing
type StageHelper struct {
	Workdir    string // path to work directory
	Next       string // next step
	SpecPrefix string // spec prefix
	LogPrefix  string // log prefix for logging
	MaxFix     int    // max number of fix attempts in fix step

	Tqueue     *queue.TaskQueue // priority queue for tracking elements to generate
	TqueuePath string           // path to Tqueue json file

	Spool     *pool.SpecPool // pool for tracking elements to be fixed
	SpoolPath string         // path to Spool json file

	Pool     *pool.SpecPool // pool for tracking already completed (maybe unfixed) elements
	PoolPath string         // path to pool json file

	SyzPool *pool.SpecPool // pool for tracking existing specifications in sysdir
}

// WriteTqueue write the Tqueue field to Tqueue file
func (sh *StageHelper) WriteTqueue() error {
	data, err := sh.Tqueue.Json()
	if err != nil {
		return fmt.Errorf("failed to convert Tqueue to json: %v", err)
	}
	if err = os.WriteFile(sh.TqueuePath, []byte(data), 0644); err != nil {
		return fmt.Errorf("failed to write Tqueue file: %v", err)
	}
	return nil
}

// WriteSpool write the Spool field to Spool file
func (sh *StageHelper) WriteSpool() error {
	data, err := sh.Spool.Json()
	if err != nil {
		return fmt.Errorf("failed to convert Spool to json: %v", err)
	}
	if err = os.WriteFile(sh.SpoolPath, []byte(data), 0644); err != nil {
		return fmt.Errorf("failed to write Spool file: %v", err)
	}
	return nil
}

// WritePool write the Pool field to Pool file
func (sh *StageHelper) WritePool() error {
	data, err := sh.Pool.Json()
	if err != nil {
		return fmt.Errorf("failed to convert Pool to json: %v", err)
	}
	if err = os.WriteFile(sh.PoolPath, []byte(data), 0644); err != nil {
		return fmt.Errorf("failed to write Pool file: %v", err)
	}
	return nil
}

// WriteCurrStat write the current state of Tqueue, Spool, and Pool to files
func (sh *StageHelper) WriteCurrStat() error {
	// write current state of Tqueue, Spool, and Pool to files after each step
	if err := sh.WriteTqueue(); err != nil {
		return fmt.Errorf("failed to write Tqueue: %v", err)
	}
	if err := sh.WriteSpool(); err != nil {
		return fmt.Errorf("failed to write Spool: %v", err)
	}
	if err := sh.WritePool(); err != nil {
		return fmt.Errorf("failed to write Pool: %v", err)
	}
	return nil
}

// RecoverProgress recover existing progress from workdir
func (sh *StageHelper) RecoverProgress() error {
	// recover Tqueue field
	if _, err := os.Stat(sh.TqueuePath); err == nil {
		data, err := os.ReadFile(sh.TqueuePath)
		if err != nil {
			return fmt.Errorf("failed to read Tqueue file: %v", err)
		}
		sh.Tqueue, err = queue.NewTaskQueueFromJson(string(data))
		if err != nil {
			return fmt.Errorf("failed to recover Tqueue from json: %v", err)
		}
	}

	// recover Spool field
	if _, err := os.Stat(sh.SpoolPath); err == nil {
		data, err := os.ReadFile(sh.SpoolPath)
		if err != nil {
			return fmt.Errorf("failed to read Spool file: %v", err)
		}
		if err = json.Unmarshal(data, &sh.Spool); err != nil {
			return fmt.Errorf("failed to unmarshal Spool json: %v", err)
		}
	}

	// recover Pool field
	if _, err := os.Stat(sh.PoolPath); err == nil {
		data, err := os.ReadFile(sh.PoolPath)
		if err != nil {
			return fmt.Errorf("failed to read Pool file: %v", err)
		}
		if err = json.Unmarshal(data, &sh.Pool); err != nil {
			return fmt.Errorf("failed to unmarshal Pool json: %v", err)
		}
	}

	return nil
}

// SaveQueryMessages save the current messages of kAgent to a temp file with given prefix
func (sh *StageHelper) SaveQueryMessages(kAgent *agent.Agent, prefix string) error {
	timestamp := time.Now().UnixMilli()
	fn := fmt.Sprintf("%s%d.msg", prefix, timestamp)
	fp := filepath.Join(sh.Workdir, fn)
	f, err := os.Create(fp)
	if err != nil {
		return fmt.Errorf("failed to create file for saving messages: %v", err)
	}
	defer f.Close()
	kAgent.SaveMessage(f)
	return nil
}

// UpdateNextStep update the Next field according to the current state of StageHelper
func (sh *StageHelper) UpdateNextStep() {
	sh.Next = sh.NextStep()
}

// NextStep return the next step to execute according to the current state of StageHelper.
func (sh *StageHelper) NextStep() string {
	if sh.ShouldOutline() {
		return "outline"
	}
	if sh.ShouldComplete() {
		return "complete"
	}
	if sh.ShouldFix() {
		return "fix"
	}
	// TODO: not sure if there are any omissions
	return "generate"
}

// ShouldOutline return whether the outline step should be executed, which is true
// only when Tqueue is nil, i.e. no progress has been made
func (sh *StageHelper) ShouldOutline() bool {
	return sh.Tqueue == nil
}

// ShouldComplete return whether the complete step should be executed, which is true
// when:
//   - both Tqueue and Spool are empty; or
//   - all elements in Tqueue and Spool are already in Pool
func (sh *StageHelper) ShouldComplete() bool {
	if sh.Tqueue.Empty() && sh.Spool.Empty() {
		return true
	}
	allInPool := true
	for _, te := range sh.Tqueue.Slice() {
		if !sh.Pool.Exists(te.Name) {
			allInPool = false
			break
		}
	}
	if allInPool {
		for _, se := range *sh.Spool {
			if !sh.Pool.Exists(se.Name) {
				allInPool = false
				break
			}
		}
	}
	return allInPool
}

// ShouldFix return whether the fix step should be executed, which is true when:
//   - Tqueue is empty and Spool is not empty; or
//   - syscall in Spool and top element in Tqueue is syscall
func (sh *StageHelper) ShouldFix() bool {
	if sh.Tqueue.Empty() && !sh.Spool.Empty() {
		return true
	}
	hasSyscall := false
	for _, se := range *sh.Spool {
		if se.Type == queue.TaskHeapElemTypeSyscall.String() {
			hasSyscall = true
		}
	}
	topElem, err := sh.Tqueue.Peek()
	return hasSyscall && (sh.Tqueue.Empty() || (err == nil && topElem.Type == queue.TaskHeapElemTypeSyscall.String()))
}

// UpdatePool update the Pool field with elements from given SpecPool
func (sh *StageHelper) UpdatePool(sq *pool.SpecPool) {
	for _, se := range *sq {
		// if se.Name not in sh.Pool, add it to the pool directly
		if !sh.Pool.Exists(se.Name) {
			sh.Pool.Insert(*se)
			continue
		}
		// if se.Name already in sh.Pool, update it only if se is valid and sh.Pool[se.Name] is invalid
		pse, err := sh.Pool.Get(se.Name)
		if err != nil {
			return
		}
		if !pse.Valid && se.Valid {
			sh.Pool.Insert(*se)
		}
	}
}
