package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kimjooyoon/gooo-improvement-proposer/internal/proposer"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}
	switch args[0] {
	case "compile":
		return compile(args[1:], stdout, stderr)
	case "propose":
		return propose(args[1:], stdout, stderr)
	case "conformance":
		return conformance(args[1:], stdout, stderr)
	case "version":
		fmt.Fprintln(stdout, "gooo-improvement-proposer/v0.1.0")
		return 0
	default:
		usage(stderr)
		return 2
	}
}

func usage(writer io.Writer) {
	fmt.Fprintln(writer, "usage: gooo-improvement-proposer <compile|propose|conformance|version>")
}

func compile(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("compile", flag.ContinueOnError)
	flags.SetOutput(stderr)
	sourcePath := flags.String("source", "examples/improvement-proposer.gooo", "Gooo source path")
	contractPath := flags.String("contract", "contracts/improvement-proposer-denominator-v1.json", "fixed denominator path")
	outputIR := flags.String("output-ir", "", "caller-owned absolute semantic IR output")
	outputEvaluator := flags.String("output-evaluator", "", "caller-owned absolute generated evaluator output")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !absolute(*outputIR) || !absolute(*outputEvaluator) {
		fmt.Fprintln(stderr, "compile requires absolute -output-ir and -output-evaluator paths")
		return 2
	}
	meta, err := compileMeta(*sourcePath, *contractPath)
	if err != nil {
		fmt.Fprintf(stderr, "compile: %v\n", err)
		return 1
	}
	if err := proposer.WriteBytes(*outputIR, meta.irRaw); err != nil {
		fmt.Fprintf(stderr, "write semantic IR: %v\n", err)
		return 1
	}
	if err := proposer.WriteBytes(*outputEvaluator, meta.generatedRaw); err != nil {
		fmt.Fprintf(stderr, "write generated evaluator: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "compiled semantic_ir=%s generated_evaluator=%s\n", *outputIR, *outputEvaluator)
	return 0
}

func propose(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("propose", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", ".", "input repository root")
	sourcePath := flags.String("source", "examples/improvement-proposer.gooo", "Gooo source path")
	contractPath := flags.String("contract", "contracts/improvement-proposer-denominator-v1.json", "fixed denominator path")
	inputPath := flags.String("input", "", "exact ledger/capability/counterexample/utility input")
	outputDir := flags.String("output-dir", "", "caller-owned absolute output directory")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *inputPath == "" || !absolute(*outputDir) {
		fmt.Fprintln(stderr, "propose requires -input and absolute -output-dir")
		return 2
	}
	evaluation, err := loadAndEvaluate(*root, *sourcePath, *contractPath, *inputPath)
	if err != nil {
		fmt.Fprintf(stderr, "propose: %v\n", err)
		return 1
	}
	if err := proposer.WriteEvaluation(*outputDir, evaluation); err != nil {
		fmt.Fprintf(stderr, "write proposal: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "%s state=%s candidates=%d output=%s\n", evaluation.Proposal.CaseID, evaluation.Proposal.State, evaluation.Proposal.CandidateCount, *outputDir)
	return 0
}

func conformance(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("conformance", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", ".", "input repository root")
	sourcePath := flags.String("source", "examples/improvement-proposer.gooo", "Gooo source path")
	contractPath := flags.String("contract", "contracts/improvement-proposer-denominator-v1.json", "fixed denominator path")
	corpusPath := flags.String("corpus", "fixtures/corpus.json", "conformance corpus")
	outputDir := flags.String("output-dir", "", "caller-owned absolute output directory")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !absolute(*outputDir) {
		fmt.Fprintln(stderr, "conformance requires absolute -output-dir")
		return 2
	}
	if err := proposer.EnsureCallerOwnedOutput(*outputDir); err != nil {
		fmt.Fprintf(stderr, "conformance output: %v\n", err)
		return 1
	}
	corpusRaw, err := os.ReadFile(*corpusPath)
	if err != nil {
		fmt.Fprintf(stderr, "read corpus: %v\n", err)
		return 1
	}
	var corpus corpus
	decoder := json.NewDecoder(strings.NewReader(string(corpusRaw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&corpus); err != nil {
		fmt.Fprintf(stderr, "decode corpus: %v\n", err)
		return 1
	}
	if corpus.Schema != proposer.ProtocolSchema+"/corpus/v1" || corpus.DenominatorID == "" || corpus.Total != 9 || len(corpus.Cases) != 9 {
		fmt.Fprintln(stderr, "corpus must contain exactly nine cases")
		return 1
	}
	states := map[string]int{"normal": 0, "unknown": 0, "refuted": 0}
	seen := map[string]bool{}
	index := conformanceIndex{Schema: proposer.ProtocolSchema + "/conformance/v1", CorpusID: corpus.CorpusID, Total: corpus.Total, Cases: make([]conformanceCase, 0, len(corpus.Cases))}
	for _, item := range corpus.Cases {
		if seen[item.CaseID] {
			fmt.Fprintf(stderr, "duplicate corpus case %s\n", item.CaseID)
			return 1
		}
		seen[item.CaseID] = true
		if item.Class != "normal" && item.Class != "unknown" && item.Class != "refuted" {
			fmt.Fprintf(stderr, "invalid corpus class %s\n", item.CaseID)
			return 1
		}
		states[item.Class]++
		inputPath := filepath.Join(*root, item.Input)
		first, err := loadAndEvaluate(*root, *sourcePath, *contractPath, inputPath)
		if err != nil {
			fmt.Fprintf(stderr, "%s first evaluation: %v\n", item.CaseID, err)
			return 1
		}
		second, err := loadAndEvaluate(*root, *sourcePath, *contractPath, inputPath)
		if err != nil {
			fmt.Fprintf(stderr, "%s replay evaluation: %v\n", item.CaseID, err)
			return 1
		}
		if !sameEvaluation(first, second) {
			fmt.Fprintf(stderr, "%s: deterministic replay mismatch\n", item.CaseID)
			return 1
		}
		if first.Proposal.State != proposer.State(item.ExpectedState) || first.Proposal.CandidateCount != item.ExpectedCandidateCount || first.Proposal.BlockedFrontier != item.ExpectedBlockedFrontier {
			fmt.Fprintf(stderr, "%s: expectation mismatch state=%s candidates=%d frontier=%d\n", item.CaseID, first.Proposal.State, first.Proposal.CandidateCount, first.Proposal.BlockedFrontier)
			return 1
		}
		caseDir := filepath.Join(*outputDir, item.CaseID)
		if err := proposer.WriteEvaluation(caseDir, first); err != nil {
			fmt.Fprintf(stderr, "%s write: %v\n", item.CaseID, err)
			return 1
		}
		index.Cases = append(index.Cases, conformanceCase{Ordinal: item.Ordinal, CaseID: item.CaseID, Class: item.Class, State: string(first.Proposal.State), CandidateCount: first.Proposal.CandidateCount, EvidenceEdges: first.Proposal.EvidenceEdges, BlockedFrontier: first.Proposal.BlockedFrontier, OutputFiles: len(proposer.FixedOutputFiles)})
		fmt.Fprintf(stdout, "%s class=%s state=%s candidates=%d\n", item.CaseID, item.Class, first.Proposal.State, first.Proposal.CandidateCount)
	}
	if states["normal"] < 3 || states["unknown"] < 3 || states["refuted"] < 3 {
		fmt.Fprintln(stderr, "corpus requires at least three normal, unknown, and refuted cases")
		return 1
	}
	index.States = states
	if _, err := proposer.WriteJSON(filepath.Join(*outputDir, "conformance-index.json"), index); err != nil {
		fmt.Fprintf(stderr, "write conformance index: %v\n", err)
		return 1
	}
	return 0
}

type compiledMeta struct {
	ir           proposer.SemanticIR
	irRaw        []byte
	generatedRaw []byte
	contract     proposer.Contract
}

func compileMeta(sourcePath, contractPath string) (compiledMeta, error) {
	sourceRaw, err := os.ReadFile(sourcePath)
	if err != nil {
		return compiledMeta{}, err
	}
	contract, contractRaw, err := proposer.LoadContract(contractPath)
	if err != nil {
		return compiledMeta{}, err
	}
	contractDigest := proposer.DigestBytes(contractRaw)
	ir, err := proposer.CompileSource(sourcePath, sourceRaw, contract, contractDigest)
	if err != nil {
		return compiledMeta{}, err
	}
	irRaw, err := proposer.SemanticIRBytes(ir)
	if err != nil {
		return compiledMeta{}, err
	}
	generatedRaw := proposer.GenerateEvaluator(ir, proposer.DigestBytes(irRaw), contractDigest)
	return compiledMeta{ir: ir, irRaw: irRaw, generatedRaw: generatedRaw, contract: contract}, nil
}

func loadAndEvaluate(root, sourcePath, contractPath, inputPath string) (proposer.Evaluation, error) {
	meta, err := compileMeta(sourcePath, contractPath)
	if err != nil {
		return proposer.Evaluation{}, err
	}
	inputRaw, err := os.ReadFile(inputPath)
	if err != nil {
		return proposer.Evaluation{}, err
	}
	var input proposer.Input
	decoder := json.NewDecoder(bytes.NewReader(inputRaw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return proposer.Evaluation{}, err
	}
	evaluatorRaw, err := os.ReadFile(filepath.Join(root, "internal", "proposer", "evaluator.go"))
	if err != nil {
		return proposer.Evaluation{}, err
	}
	return proposer.Propose(input, proposer.DigestBytes(inputRaw), meta.ir, meta.irRaw, meta.generatedRaw, proposer.DigestBytes(mustRead(contractPath)), proposer.DigestBytes(evaluatorRaw))
}

func mustRead(path string) []byte {
	data, _ := os.ReadFile(path)
	return data
}

func sameEvaluation(left, right proposer.Evaluation) bool {
	leftProposal, _ := json.Marshal(left.Proposal)
	rightProposal, _ := json.Marshal(right.Proposal)
	return bytes.Equal(leftProposal, rightProposal) && bytes.Equal(left.SemanticIRRaw, right.SemanticIRRaw) && bytes.Equal(left.GeneratedRaw, right.GeneratedRaw) && bytes.Equal(proposerEvents(left.Events), proposerEvents(right.Events)) && left.Dossier == right.Dossier && sameReplay(left.Replay, right.Replay)
}

func proposerEvents(events []proposer.CandidateEvent) []byte {
	data, _ := json.Marshal(events)
	return data
}

func sameReplay(left, right proposer.ReplayReceipt) bool {
	leftRaw, _ := json.Marshal(left)
	rightRaw, _ := json.Marshal(right)
	return bytes.Equal(leftRaw, rightRaw)
}

func absolute(path string) bool {
	return path != "" && filepath.IsAbs(path)
}

type corpus struct {
	Schema        string       `json:"schema"`
	CorpusID      string       `json:"corpus_id"`
	DenominatorID string       `json:"denominator_id"`
	Total         int          `json:"total"`
	Cases         []corpusCase `json:"cases"`
}

type corpusCase struct {
	Ordinal                 int    `json:"ordinal"`
	CaseID                  string `json:"case_id"`
	Class                   string `json:"class"`
	Input                   string `json:"input"`
	ExpectedState           string `json:"expected_state"`
	ExpectedCandidateCount  int    `json:"expected_candidate_count"`
	ExpectedBlockedFrontier int    `json:"expected_blocked_frontier"`
}

type conformanceIndex struct {
	Schema   string            `json:"schema"`
	CorpusID string            `json:"corpus_id"`
	Total    int               `json:"total"`
	Cases    []conformanceCase `json:"cases"`
	States   map[string]int    `json:"states"`
}

type conformanceCase struct {
	Ordinal         int    `json:"ordinal"`
	CaseID          string `json:"case_id"`
	Class           string `json:"class"`
	State           string `json:"state"`
	CandidateCount  int    `json:"candidate_count"`
	EvidenceEdges   int    `json:"evidence_edges"`
	BlockedFrontier int    `json:"blocked_frontier"`
	OutputFiles     int    `json:"output_files"`
}
