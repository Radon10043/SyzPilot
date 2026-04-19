package main

import (
	"bytes"
	"cmp"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/aclements/go-moremath/stats"
	"github.com/jedib0t/go-pretty/v6/table"
)

var (
	flagExpdir string
	flagMaster string
	flagFormat string
	flagStrict bool
)

type Kernel struct {
	Name      string
	Version   string
	Subsystem string
}

type Fuzzer struct {
	Name   string
	Master bool
}

type Result struct {
	// coverage achieved by Fuzzer
	Coverage []int
	// average coverage
	AvgCov int
	// A12 value compare with master, i.e. what the probability
	// that master has higher coverage than current fuzzer
	CovA12 float64
	// p-value of the coverage compared with master
	CovPV float64
	// number of unique crashes triggered by Fuzzer
	Crashes []int
	// average number of unique crashes found by Fuzzer
	AvgCra float64
	// A12 value compare with master, i.e. what the probability
	// that master has more crashes than current fuzzer
	CraA12 float64
	// p-value of the crashes compared with master
	CraPV float64
}

type Expinfo struct {
	Kernel Kernel
	Fuzzer Fuzzer
	Result Result
}

func main() {
	flag.StringVar(&flagExpdir, "expdir", "", "path to the experiment directory")
	flag.StringVar(&flagMaster, "master", "", "master fuzzer")
	flag.StringVar(&flagFormat, "format", "text", "output format, support text, csv, tsv, markdown, and html")
	flag.BoolVar(&flagStrict, "strict", false, "strict mode")
	flag.Parse()

	if err := checkExpdir(flagExpdir); err != nil {
		panic(err)
	}

	expdir, err := filepath.Abs(flagExpdir)
	if err != nil {
		panic(err)
	}

	entries, err := os.ReadDir(expdir)
	if err != nil {
		panic(err)
	}

	// collect experiment info
	eiSlice := make([]Expinfo, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		eis, err := parseFuzzing(expdir, entry)
		if err != nil {
			panic(err)
		}
		eiSlice = append(eiSlice, *eis...)
	}

	// perform statistical test and update the result in place
	statTest(&eiSlice)

	// print the result in a table format
	printTable(eiSlice)
}

// calculate mann-whitney u test and A12 for coverage and crashes
func statTest(eis *[]Expinfo) {
	// create fuzzer set, kernel set, and a map from fuzzer and kernel to result
	eimap := make(map[Fuzzer]map[Kernel]*Result)
	fuzzerMap := make(map[Fuzzer]bool, 0)
	kernMap := make(map[Kernel]bool, 0)
	var master *Fuzzer = nil
	for i := range *eis {
		ei := &(*eis)[i]
		if ei.Fuzzer.Master {
			master = &ei.Fuzzer
		}
		if _, ok := eimap[ei.Fuzzer]; !ok {
			eimap[ei.Fuzzer] = make(map[Kernel]*Result)
			fuzzerMap[ei.Fuzzer] = true
		}
		if _, ok := eimap[ei.Fuzzer][ei.Kernel]; !ok {
			kernMap[ei.Kernel] = true
		}
		eimap[ei.Fuzzer][ei.Kernel] = &ei.Result
	}

	// if no master fuzzer, statistical test is meaningless
	if master == nil {
		fmt.Fprintln(os.Stderr, "no master fuzzer found, skip statistical test")
		return
	}

	// create slices for fuzzers and kernels
	fuzzers := make([]Fuzzer, 0)
	for fuzzer := range fuzzerMap {
		fuzzers = append(fuzzers, fuzzer)
	}
	kernels := make([]Kernel, 0)
	for kernel := range kernMap {
		kernels = append(kernels, kernel)
	}

	// calculate A12 and p-value for each fuzzer and kernel
	for _, fuzzer := range fuzzers {
		if fuzzer.Master {
			continue
		}
		for _, kernel := range kernels {
			masterRes, ok1 := eimap[*master][kernel]
			slaveRes, ok2 := eimap[fuzzer][kernel]
			if !ok1 || !ok2 {
				continue
			}
			statTestImpl(masterRes, slaveRes)
		}
	}
}

// statTestImpl calculates the A12 value and p-value for coverage and crashes, and updates the result in place
func statTestImpl(res1 *Result, res2 *Result) {
	res2.CovA12 = a12(toFloat64Slice(res1.Coverage), toFloat64Slice(res2.Coverage))
	res2.CovPV = utest(toFloat64Slice(res1.Coverage), toFloat64Slice(res2.Coverage))
	res2.CraA12 = a12(toFloat64Slice(res1.Crashes), toFloat64Slice(res2.Crashes))
	res2.CraPV = utest(toFloat64Slice(res1.Crashes), toFloat64Slice(res2.Crashes))
}

func toFloat64Slice(intSlice []int) []float64 {
	floatSlice := make([]float64, len(intSlice))
	for i, v := range intSlice {
		floatSlice[i] = float64(v)
	}
	return floatSlice
}

// utest performs the Mann-Whitney U test on two slices of integers, returns p-value
func utest(v1 []float64, v2 []float64) float64 {
	res, err := stats.MannWhitneyUTest(v1, v2, stats.LocationDiffers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "perform mann-whitney u test failed: %v\n", err)
		return -1
	}
	return res.P
}

// a12 calculates the A12 value of two slices of integers, returns the A12 value
func a12(v1 []float64, v2 []float64) float64 {
	res, err := stats.MannWhitneyUTest(v1, v2, stats.LocationDiffers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "calculate a12 failed: %v\n", err)
		return -1
	}
	return res.U / float64(res.N1*res.N2)
}

// printTable prints the experiment info in a table format
func printTable(eiSlice []Expinfo) {
	slices.SortFunc(eiSlice, func(a, b Expinfo) int {
		aIsKernel := a.Kernel.Subsystem == "kernel"
		bIsKernel := b.Kernel.Subsystem == "kernel"
		if aIsKernel && !bIsKernel {
			return -1
		}
		if !aIsKernel && bIsKernel {
			return 1
		}
		aFullInfo := fmt.Sprintf("%s-%s-%s-%s", a.Kernel.Name, a.Kernel.Subsystem, a.Kernel.Version, a.Fuzzer.Name)
		bFullInfo := fmt.Sprintf("%s-%s-%s-%s", b.Kernel.Name, b.Kernel.Subsystem, b.Kernel.Version, b.Fuzzer.Name)
		return cmp.Compare(aFullInfo, bFullInfo)
	})
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{
		"Kernel", "Version", "Subsystem", "Fuzzer",
		"Avg. Coverage", "CovA12", "CovPV",
		"Avg. Crashes", "CraA12", "CraPV"},
	)
	for _, ei := range eiSlice {
		t.AppendRow(table.Row{
			ei.Kernel.Name,
			ei.Kernel.Version,
			ei.Kernel.Subsystem,
			ei.Fuzzer.Name,
			ei.Result.AvgCov,
			fmt.Sprintf("%.4f", ei.Result.CovA12),
			fmt.Sprintf("%.4f", ei.Result.CovPV),
			ei.Result.AvgCra,
			fmt.Sprintf("%.4f", ei.Result.CraA12),
			fmt.Sprintf("%.4f", ei.Result.CraPV),
		})
	}

	switch flagFormat {
	case "text":
		t.Render()
	case "csv":
		t.RenderCSV()
	case "markdown":
		t.RenderMarkdown()
	case "html":
		t.RenderHTML()
	case "tsv":
		t.RenderTSV()
	default:
		panic(fmt.Sprintf("Unsupported format: %s", flagFormat))
	}
}

// checkExpdir checks the validity of the experiment directory, includes:
//   - if the experiment directory is empty
//   - directory existence
//   - if repeat times is consistent
func checkExpdir(expdir string) error {
	repeat := -1
	fuzzers := make([]string, 0)

	// check if empty
	if expdir == "" {
		return fmt.Errorf("experiment directory is empty")
	}

	// check existence
	if _, err := os.Stat(expdir); os.IsNotExist(err) {
		return fmt.Errorf("experiment directory does not exist: %s", expdir)
	}

	if !flagStrict {
		return nil
	}

	// follwoing are strict checks
	// check repeat times consistency
	entries, err := os.ReadDir(expdir)
	if err != nil {
		return fmt.Errorf("failed to read experiment dir: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		fuzzers = append(fuzzers, entry.Name())
		subentries, err := os.ReadDir(filepath.Join(expdir, entry.Name()))
		if err != nil {
			return fmt.Errorf("failed to read fuzzer dir: %w", err)
		}
		for _, subentry := range subentries {
			if !subentry.IsDir() {
				continue
			}
			matches, err := filepath.Glob(
				filepath.Join(
					expdir, entry.Name(),
					subentry.Name(),
					"*",
					"fuzz.log",
				),
			)
			if err != nil {
				return fmt.Errorf("failed to find fuzz.log files: %w", err)
			}
			if repeat == -1 {
				repeat = len(matches)
			} else if repeat != len(matches) {
				return fmt.Errorf(
					"inconsistent repeat times: previous is %d vs %s is %d",
					repeat,
					filepath.Join(expdir, entry.Name(), subentry.Name()),
					len(matches),
				)
			}
		}
	}
	return nil
}

// parseFuzzing parses the fuzzing results of a fuzzer, and returns a list of Expinfo
func parseFuzzing(expdir string, dentry os.DirEntry) (*[]Expinfo, error) {
	fuzzer := Fuzzer{
		Name:   dentry.Name(),
		Master: dentry.Name() == flagMaster,
	}
	eiSlice := make([]Expinfo, 0)

	// parse fuzzing result
	entries, err := os.ReadDir(filepath.Join(expdir, dentry.Name()))
	if err != nil {
		return nil, fmt.Errorf("failed to read fuzzing dir: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		entSlice := strings.Split(entry.Name(), "-")
		if len(entSlice) != 3 {
			return nil, fmt.Errorf("invalid dir name: %s", entry.Name())
		}
		kernel := Kernel{
			Name:      entSlice[0],
			Version:   entSlice[1],
			Subsystem: entSlice[2],
		}
		ei, err := parseFuzzingResult(expdir, fuzzer, kernel)
		if err != nil {
			return nil, fmt.Errorf("failed to parse fuzzing result: %w", err)
		}
		eiSlice = append(eiSlice, Expinfo{
			Kernel: kernel,
			Fuzzer: fuzzer,
			Result: *ei,
		})
	}

	return &eiSlice, nil
}

// parseFuzzingResult parses the fuzzing results of a fuzzer on a kernel, and returns the Result
func parseFuzzingResult(expdir string, fuzzer Fuzzer, kernel Kernel) (*Result, error) {
	// find fuzzing log files
	outRoot := filepath.Join(
		expdir,
		fuzzer.Name,
		kernel.Name+"-"+kernel.Version+"-"+kernel.Subsystem,
	)

	// collect coverage
	pattern := filepath.Join(outRoot, "*", "fuzz.log")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to find fuzz.log files: %w", err)
	}
	repeat := len(matches)
	if repeat == 0 {
		return nil, fmt.Errorf("failed to find fuzz.log in %s", outRoot)
	}
	coverage := make([]int, 0)
	totalCov := 0
	for _, mat := range matches {
		cov, err := parseCoverage(mat)
		if err != nil {
			return nil, fmt.Errorf("failed to parse coverage: %w", err)
		}
		coverage = append(coverage, cov)
		totalCov += cov
	}

	// collect crashes
	crashes := make([]int, 0)
	totalCra := 0
	matches, err = filepath.Glob(filepath.Join(outRoot, "*", "crashes"))
	if err != nil {
		return nil, fmt.Errorf("failed to find crashes files: %w", err)
	}
	for _, mat := range matches {
		cra, err := parseCrashes(mat)
		if err != nil {
			return nil, fmt.Errorf("failed to parse cra: %w", err)
		}
		crashes = append(crashes, cra)
		totalCra += cra
	}

	return &Result{
		Coverage: coverage,
		AvgCov:   totalCov / repeat,
		CovA12:   -1,
		CovPV:    -1,
		Crashes:  crashes,
		AvgCra:   float64(totalCra) / float64(repeat),
		CraA12:   -1,
		CraPV:    -1,
	}, nil
}

// parseCrashes parses the number of crashes from a crashes directory, and returns the number of crashes
func parseCrashes(fpath string) (int, error) {
	entries, err := os.ReadDir(fpath)
	if err != nil {
		return -1, fmt.Errorf("failed to read crashes dir: %w", err)
	}
	return len(entries), nil
}

// parseCoverage parses the coverage from a fuzzing log file, and returns the coverage value
func parseCoverage(fpath string) (int, error) {
	cov := 0
	b, err := os.ReadFile(fpath)
	if err != nil {
		return -1, fmt.Errorf("failed to read file: %w", err)
	}
	bSlice := bytes.Split(b, []byte("\n"))
	for i := len(bSlice) - 1; i >= 0; i-- {
		bLine := bSlice[i]
		if !bytes.Contains(bLine, []byte("coverage=")) {
			continue
		}
		tmp := bytes.Split(bLine, []byte("coverage="))
		bCov := bytes.Split(tmp[1], []byte(" "))[0]
		cov, err = strconv.Atoi(string(bCov))
		if err != nil {
			return -1, fmt.Errorf("failed to parse coverage: %w", err)
		}
		break
	}
	return cov, nil
}
