package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/Radon10043/cloud/src/pkg/agent"
	"github.com/Radon10043/cloud/src/pkg/database"
	osu "github.com/Radon10043/cloud/src/pkg/osutil"
	"github.com/Radon10043/cloud/src/pkg/pool"
	"github.com/Radon10043/cloud/src/pkg/queue"
	"github.com/Radon10043/cloud/src/pkg/stage"
	"github.com/joho/godotenv"
	"github.com/tmc/langchaingo/llms/openai"
)

var (
	flagDb     string
	flagModel  string
	flagPrompt string
	flagEnv    string
	flagSysdir string
	flagOutdir string
	flagOs     string
)

var (
	keyMap = map[osu.OsType][]string{
		osu.Linux: {
			".ioctl", ".unlocked_ioctl", ".compat_ioctl", ".mmap", ".uring_cmd",
			".setsockopt", ".getsockopt", ".recvmsg", ".sendmsg",
		},
		osu.FreeBSD: {
			".d_ioctl", ".d_open", ".d_read", ".d_write", ".d_mmap", ".d_poll",
			".vop_ioctl", ".vop_lookup", ".vop_create", ".vop_mkdir", ".vop_setattr",
			".vop_getextattr", ".vop_setextattr", ".pru_control", ".pru_attach",
			".pru_bind", ".pru_connect", ".pru_send", ".pru_rcvd", ".ph_type",
		},
		osu.OpenBSD: {
			".d_ioctl", ".d_open", ".d_read", ".d_write", ".d_mmap", ".d_poll", ".d_kqfilter",
			".fo_ioctl", ".fo_read", ".fo_write", ".fo_poll", ".fo_kqfilter",
			".vop_ioctl", ".vop_read", ".vop_write", ".vop_lookup", ".vop_create", ".vop_mkdir", ".vop_setattr",
			".pru_control", ".pru_attach", ".pru_bind", ".pru_connect", ".pru_send", ".pru_rcvd",
			".ioctl", ".mmap", ".unlocked_ioctl",
		},
	}
)

func main() {
	flag.StringVar(&flagDb, "db", "", "Path to the database file")
	flag.StringVar(&flagModel, "model", "", "The model to use")
	flag.StringVar(
		&flagPrompt,
		"prompt",
		"./data/prompts/outline/instruction.md,"+
			"./data/prompts/outline/example_media.md,"+
			"./data/prompts/outline/example_ppp.md",
		"Path to the outline system prompt file(s), use comma to separate multiple files",
	)
	flag.StringVar(&flagEnv, "env", ".env", "Path to the .env file")
	flag.StringVar(&flagSysdir, "sysdir", "./syzkaller/sys", "Path to the syzkaller like sys directory")
	flag.StringVar(&flagOutdir, "outdir", "", "Path to the output directory")
	flag.StringVar(&flagOs, "os", runtime.GOOS, "os type")
	flag.Parse()

	// load environment variables
	if err := godotenv.Load(flagEnv); err != nil {
		log.Fatalf("Failed to load environment variables: %v", err)
	}

	// connect to database
	db := database.Database{Path: flagDb}
	err := db.Connect()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// get all global variables, extract variables that contain keys and deduplicate them
	gvs, err := db.GetAllGlobalVar()
	if err != nil {
		panic(err)
	}
	log.Printf("Total global variables: %d\n", len(gvs))
	osType, err := osu.ParseOsType(flagOs)
	if err != nil {
		log.Fatalf("Failed to parse OS type: %v", err)
	}
	keyGvs := extract(gvs, osType)
	keyGvs = deduplicate(keyGvs)
	slices.SortFunc(keyGvs, func(a, b database.GlobalVar) int {
		return strings.Compare(a.Name, b.Name)
	})
	log.Printf("Total key global variables: %d\n", len(keyGvs))

	// create prompt for spec outline generation
	prompt := createPrompt()

	// create agent
	llm, err := openai.New(
		openai.WithBaseURL(os.Getenv("OPENAI_BASE_URL")),
		openai.WithToken(os.Getenv("OPENAI_API_KEY")),
		openai.WithModel(flagModel),
	)
	if err != nil {
		panic(err)
	}
	kAgent := agent.NewAgent(&db, llm)

	// create syz pool from sysdir
	syzPool, err := createPoolFromSysdir(flagSysdir)
	if err != nil {
		panic(err)
	}

	// set to deduplicate syscalls in the tasks
	syscallSet := recoverSyscallSet()

	// create output directory
	err = os.MkdirAll(flagOutdir, 0755)
	if err != nil {
		panic(err)
	}

	// generate outlines for each key global variable, the artifacts under
	// outdir can also be used for spec generation
	for i, gv := range keyGvs {
		specdir := filepath.Join(flagOutdir, "specs", gv.Name+"#"+flagModel)
		logger := log.New(
			os.Stdout,
			fmt.Sprintf("[%d/%d][%s] ", i+1, len(keyGvs), gv.Name),
			log.LstdFlags|log.Lmsgprefix,
		)

		err := os.MkdirAll(specdir, 0755)
		if err != nil {
			panic(err)
		}

		var sh *stage.StageHelper = &stage.StageHelper{
			Workdir:    specdir,
			Next:       "outline",
			Tqueue:     nil,
			TqueuePath: filepath.Join(specdir, ".tqueue"),
		}
		_, err = os.Stat(sh.TqueuePath)
		if err == nil { // Tqueue file exists, skip this variable
			logger.Printf(".tqueue already exists, skip outline generation\n")
			continue
		}

		// sleep for a while to avoid frequent requests
		slpTime := rand.Int31n(11) + 10
		logger.Printf("sleep for %d seconds to avoid frequent requests ...\n", slpTime)
		time.Sleep(time.Duration(slpTime) * time.Second)

		err = stage.ExecOutlineStep(kAgent, prompt, &gv, logger, sh)
		if err != nil {
			fmt.Printf("failed to generate outline for %s, skip it: %v\n", gv.Name, err)
			continue
		}
		ntq := dedupTaskQueue(sh.Tqueue, syscallSet, syzPool)

		for _, elem := range ntq.Slice() {
			syscallSet[elem.Name] = true
		}
		sh.Tqueue = ntq
		sh.WriteTqueue()
	}
	log.Println("minitask generated successfully.")
}

// recoverSyscallSet recovers the syscall set from the existing .tqueue files under the output directory,
func recoverSyscallSet() map[string]bool {
	syscallSet := make(map[string]bool)
	pattern := filepath.Join(flagOutdir, "specs", "*", ".tqueue")
	files, _ := filepath.Glob(pattern)
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		tq, err := queue.NewTaskQueueFromJson(string(data))
		if err != nil {
			continue
		}
		for _, elem := range tq.Slice() {
			syscallSet[elem.Name] = true
		}
	}
	return syscallSet
}

// dedupTaskQueue deduplicate tasks from the task queue based on the syscallSet and syzPool,
// and return a new task queue without duplicates
func dedupTaskQueue(tq *queue.TaskQueue, syscallSet map[string]bool, syzPool *pool.SpecPool) *queue.TaskQueue {
	ntq := queue.NewTaskQueue()
	for _, elem := range tq.Slice() {
		if needDedup(&elem, syscallSet, syzPool) {
			continue
		}
		ntq.Push(&elem)
	}
	return ntq
}

// needDedup checks if the task queue element needs to be deduplicatedbased on its type and name,
// and the existing syscallSet and syzPool
func needDedup(elem *queue.TaskQueueElem, syscallSet map[string]bool, syzPool *pool.SpecPool) bool {
	if elem.Type == queue.TaskHeapElemTypeInitSyscall.String() {
		// for init syscall, it needn't dedup since we need it for ensuring consistency
		return false
	}

	// for other elements (should be syscall), it need dedup when it is already in syscallSet or syzPool
	if _, ok := syscallSet[elem.Name]; ok {
		return true
	}
	return syzPool.Exists(elem.Name)
}

// createPoolFromSysdir reads spec files from the given sysdir, convert them to
// SpecPool and merge them into a single SpecPool, return the merged SpecPool
// and error if any
func createPoolFromSysdir(sysdir string) (*pool.SpecPool, error) {
	p := &pool.SpecPool{}
	osType, err := osu.ParseOsType(flagOs)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OS type: %v", err)
	}
	fs, err := filepath.Glob(
		filepath.Join(
			sysdir,
			osType.String(),
			"*.txt",
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to glob spec files in sysdir: %v", err)
	}
	for _, f := range fs {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("failed to read spec file %s: %v", f, err)
		}
		tmp, err := pool.NewSpecPoolFromSyzlang(string(b))
		if err != nil {
			return nil, fmt.Errorf("failed to convert syzlang to SpecPool for file %s: %v", f, err)
		}
		p.Merge(tmp)
	}
	return p, nil
}

// createPrompt reads the prompt from the given file(s) and concatenate them into a single string
func createPrompt() string {
	var promptBuilder strings.Builder
	files := strings.SplitSeq(flagPrompt, ",")
	for file := range files {
		path, err := filepath.Abs(file)
		if err != nil {
			panic(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		promptBuilder.WriteString(string(data) + "\n")
	}
	return promptBuilder.String()
}

// deduplicate removes duplicate global variables based on their names
func deduplicate(gvs []database.GlobalVar) []database.GlobalVar {
	seen := make(map[string]struct{})
	var res []database.GlobalVar
	for _, gv := range gvs {
		if _, ok := seen[gv.Name]; !ok {
			seen[gv.Name] = struct{}{}
			res = append(res, gv)
		}
	}
	return res
}

// extract extracts global variables that contain specific keys related to the os type, and returns them as a slice
func extract(gvs []database.GlobalVar, os osu.OsType) []database.GlobalVar {
	var res []database.GlobalVar
	for _, gv := range gvs {
		if keyFound(os, gv.Code) {
			res = append(res, gv)
		}
	}
	return res
}

// keyFound check if any key is found in the code
func keyFound(os osu.OsType, code string) bool {
	for _, key := range keyMap[os] {
		if strings.Contains(code, key) {
			return true
		}
	}
	return false
}
