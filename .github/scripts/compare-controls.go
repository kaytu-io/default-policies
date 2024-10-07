package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
)

type Benchmark struct {
	ID          string              `json:"ID" yaml:"ID"`
	Title       string              `json:"Title" yaml:"Title"`
	SectionCode string              `json:"SectionCode" yaml:"SectionCode"`
	Connector   string              `json:"Connector" yaml:"Connector"`
	Description string              `json:"Description" yaml:"Description"`
	Children    []string            `json:"Children" yaml:"Children"`
	Tags        map[string][]string `json:"Tags" yaml:"Tags"`
	Enabled     bool                `json:"Enabled" yaml:"Enabled"`
	AutoAssign  bool                `json:"AutoAssign" yaml:"AutoAssign"`
	Controls    []string            `json:"Controls" yaml:"Controls"`
}

type QueryParameter struct {
	Key      string `json:"key" example:"key"`
	Required bool   `json:"required" example:"true"`
}

type Query struct {
	Engine         string           `json:"Engine" yaml:"Engine"`
	QueryToExecute string           `json:"QueryToExecute" yaml:"QueryToExecute"`
	PrimaryTable   string           `json:"PrimaryTable" yaml:"PrimaryTable"`
	ListOfTables   []string         `json:"ListOfTables" yaml:"ListOfTables"`
	Parameters     []QueryParameter `json:"Parameters" yaml:"Parameters"`
}

type Control struct {
	ID          string              `json:"ID" yaml:"ID"`
	Title       string              `json:"Title" yaml:"Title"`
	Description string              `json:"Description" yaml:"Description"`
	Query       Query               `json:"Query" yaml:"Query"`
	Connector   []string            `json:"Connector" yaml:"Connector"`
	Tags        map[string][]string `json:"Tags" yaml:"Tags"`
	Severity    string              `json:"Severity" yaml:"Severity"`
}

var (
	BenchmarksPath = os.Getenv("BENCHMARKS_PATH")
	ControlsPath   = os.Getenv("CONTROLS_PATH")

	red   = "\033[31m"
	green = "\033[32m"
	reset = "\033[0m"
	bold  = "\033[1m"
)

func main() {
	benchmarks := getBenchmarks(BenchmarksPath)
	controlsList, err := getControls(ControlsPath)
	if err != nil {
		log.Fatal(err)
	}

	exitCode := compareBenchmarks(benchmarks, controlsList)

	os.Exit(exitCode)
}

// getBenchmarks parses all benchmark YAML files and returns a map of benchmarks with control IDs and children
func getBenchmarks(root string) map[string]Benchmark {
	benchmarks := make(map[string]Benchmark)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(info.Name(), ".yaml") {
			benchmark := parseBenchmarkFile(path)
			benchmarks[benchmark.ID] = benchmark
		}
		return nil
	})

	if err != nil {
		log.Fatalf("Error walking through benchmark directory: %v", err)
	}

	return benchmarks
}

// parseBenchmarkFile parses a YAML file into a Benchmark struct
func parseBenchmarkFile(filePath string) Benchmark {
	file, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Error reading benchmark file: %v", err)
	}

	var benchmark Benchmark
	err = yaml.Unmarshal(file, &benchmark)
	if err != nil {
		log.Fatalf("Error unmarshalling benchmark YAML: %v", err)
	}

	return benchmark
}

// getControls parses all control YAML files and returns a list of control IDs
func getControls(root string) ([]string, error) {
	var controls []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(info.Name(), ".yaml") {
			control, err := parseControlFile(path)
			if err != nil {
				return err
			}
			if control.ID != "" {
				controls = append(controls, control.ID)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return controls, nil
}

// parseControlFile parses a YAML file into a Control struct
func parseControlFile(filePath string) (Control, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return Control{}, err
	}

	var control Control
	err = yaml.Unmarshal(file, &control)
	if err != nil {
		return Control{}, fmt.Errorf(fmt.Sprintf("control %s does not have the right structure", filePath))
	}

	return control, nil
}

// compareBenchmarks compares benchmarks' controls and children, and prints differences
func compareBenchmarks(benchmarks map[string]Benchmark, controlList []string) int {
	benchmarkControlSet := make(map[string]bool)
	controlSet := make(map[string]bool)
	benchmarkSet := make(map[string]bool)
	var missingControl []string
	var missingChild []string

	// Populate controlSet and benchmarkSet
	for _, control := range controlList {
		controlSet[control] = true
	}
	for id := range benchmarks {
		benchmarkSet[id] = true
	}

	// Check controls and children
	for _, benchmark := range benchmarks {
		for _, control := range benchmark.Controls {
			benchmarkControlSet[control] = true
			if !controlSet[control] {
				missingControl = append(missingControl, control)
			}
		}

		for _, child := range benchmark.Children {
			if !benchmarkSet[child] {
				missingChild = append(missingChild, child)
			}
		}
	}
	if len(missingControl) > 0 {
		fmt.Println(bold + "You need to add these controls to be able to merge to the main branch" + reset)
		for _, control := range missingControl {
			fmt.Println(string(green) + " + " + string(reset) + "Missing control " + control)
		}
	}
	if len(missingChild) > 0 {
		fmt.Println(bold + "You need to ensure all child benchmarks exist to be able to merge to the main branch" + reset)
		for _, child := range missingChild {
			fmt.Println(string(green) + " + " + string(reset) + "Missing child benchmark: " + child)
		}
	}

	var orphanedControls []string
	for control := range controlSet {
		if !benchmarkControlSet[control] {
			orphanedControls = append(orphanedControls, control)
		}
	}

	fmt.Println(bold + fmt.Sprintf("There are %d orphaned controls. These controls are not being reference by any benchmarks:", len(orphanedControls)) + reset)
	for _, control := range orphanedControls {
		fmt.Println(string(red) + " - " + string(reset) + control)
	}

	if len(missingControl) > 0 || len(missingChild) > 0 {
		return 1
	}

	return 0
}
