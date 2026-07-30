package main

import (
	"cmp"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"slices"

	"github.com/jedib0t/go-pretty/v6/table"
)

var (
	flagInput  string
	flagFormat string
)

type Dependency struct {
	Name  string
	Align bool
}

type Syscall struct {
	Name string
	Deps []Dependency
}

type Subsystem struct {
	Name     string
	Syscalls []Syscall
}

type Entry struct {
	Fuzzer    string
	Subsystem string
	Syscall   int
	Accurate  float64
}

func main() {
	flag.StringVar(&flagInput, "input", "", "path to the file for align evaluation")
	flag.StringVar(&flagFormat, "format", "text", "format of the output")
	flag.Parse()

	if flagInput == "" {
		flag.Usage()
		return
	}

	alignInfo := parseInput(flagInput)
	entries := calculateAcc(&alignInfo)
	printTable(entries)
}

func printTable(entries []Entry) {
	slices.SortFunc(entries, func(a, b Entry) int {
		ainfo := fmt.Sprintf("%s-%s", a.Subsystem, a.Fuzzer)
		binfo := fmt.Sprintf("%s-%s", b.Subsystem, b.Fuzzer)
		return cmp.Compare(ainfo, binfo)
	})
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{
		"Subsystem",
		"Fuzzer",
		"#Syscall",
		"Accurate",
	})
	for _, entry := range entries {
		t.AppendRow(table.Row{
			entry.Subsystem,
			entry.Fuzzer,
			entry.Syscall,
			fmt.Sprintf("%.4f", entry.Accurate),
		})
	}
	switch flagFormat {
	case "text":
		t.Render()
	case "markdown":
		t.RenderMarkdown()
	case "csv":
		t.RenderCSV()
	default:
		fmt.Printf("unsupported format: %s\n", flagFormat)
		return
	}
}

func calculateAcc(ai *map[string][]Subsystem) []Entry {
	var entries []Entry
	for fuzzer, subsystems := range *ai {
		for _, subsystem := range subsystems {
			nSyscall, acc := calculateSubsystemAcc(subsystem)
			entries = append(entries, Entry{
				Fuzzer:    fuzzer,
				Subsystem: subsystem.Name,
				Syscall:   nSyscall,
				Accurate:  acc,
			})
		}
	}
	return entries
}

func calculateSubsystemAcc(subsystem Subsystem) (int, float64) {
	nSyscall := len(subsystem.Syscalls)
	if nSyscall == 0 {
		panic(fmt.Sprintf("subsystem %s with no syscall", subsystem.Name))
	}

	accSum := 0.0
	for _, syscall := range subsystem.Syscalls {
		total := len(syscall.Deps)
		if total == 0 {
			panic(fmt.Sprintf("syscall %s with no dependency", syscall.Name))
		}
		aligned := 0
		for _, dep := range syscall.Deps {
			if dep.Align {
				aligned++
			}
		}
		accSum += float64(aligned) / float64(total)
	}
	return len(subsystem.Syscalls), accSum / float64(len(subsystem.Syscalls))
}

func parseInput(input string) map[string][]Subsystem {
	file, err := os.Open(input)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	alignInfo := make(map[string][]Subsystem)
	render := csv.NewReader(file)
	// skip the header
	if _, err := render.Read(); err != nil {
		panic(err)
	}
	for {
		record, err := render.Read()
		if err != nil {
			break
		}

		// csv fields: fuzzer,subsystem,syscall,kerndep,correspond_to_spec,aligned
		fuzzer := record[0]
		subsystem := record[1]
		syscall := record[2]
		kerndep := record[3]
		correspondToSpec := record[4]
		aligned := record[5] == "1"

		// initialize the fuzzer in alignInfo if not exist
		if _, ok := alignInfo[fuzzer]; !ok {
			alignInfo[fuzzer] = []Subsystem{}
		}

		// find the target subsystem in alignInfo, if not exist, create a new one
		subsystems := alignInfo[fuzzer]
		var targetSubsystem *Subsystem
		for i, s := range subsystems {
			if s.Name == subsystem {
				targetSubsystem = &subsystems[i]
				break
			}
		}
		if targetSubsystem == nil {
			targetSubsystem = &Subsystem{Name: subsystem}
			alignInfo[fuzzer] = append(alignInfo[fuzzer], *targetSubsystem)
			targetSubsystem = &alignInfo[fuzzer][len(alignInfo[fuzzer])-1]
		}

		// find the target syscall in the target subsystem, if not exist, create a new one
		syscalls := targetSubsystem.Syscalls
		var targetSyscall *Syscall
		for i, s := range syscalls {
			if s.Name == syscall {
				targetSyscall = &syscalls[i]
				break
			}
		}
		if targetSyscall == nil {
			targetSyscall = &Syscall{Name: syscall}
			targetSubsystem.Syscalls = append(targetSubsystem.Syscalls, *targetSyscall)
			targetSyscall = &targetSubsystem.Syscalls[len(targetSubsystem.Syscalls)-1]
		}

		// append the dependency to the target syscall
		targetSyscall.Deps = append(targetSyscall.Deps, Dependency{
			Name:  kerndep + " -> " + correspondToSpec,
			Align: aligned,
		})
	}

	return alignInfo
}
