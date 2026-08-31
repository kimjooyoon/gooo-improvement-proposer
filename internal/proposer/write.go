package proposer

import (
	"fmt"
	"os"
	"path/filepath"
)

func WriteEvaluation(outputDir string, evaluation Evaluation) error {
	if err := EnsureCallerOwnedOutput(outputDir); err != nil {
		return err
	}
	proposalRaw, err := jsonMarshalIndent(evaluation.Proposal)
	if err != nil {
		return err
	}
	replayRaw, err := jsonMarshalIndent(evaluation.Replay)
	if err != nil {
		return err
	}
	eventBytes := eventsNDJSON(evaluation.Events)
	files := map[string][]byte{
		"proposal.json":           proposalRaw,
		"candidate-events.ndjson": eventBytes,
		"semantic-ir.json":        evaluation.SemanticIRRaw,
		"generated-evaluator.go":  evaluation.GeneratedRaw,
		"replay-receipt.json":     replayRaw,
		"human-dossier.md":        []byte(evaluation.Dossier),
	}
	for _, name := range FixedOutputFiles {
		data, ok := files[name]
		if !ok {
			return fmt.Errorf("missing fixed output %s", name)
		}
		if err := WriteBytes(filepath.Join(outputDir, name), data); err != nil {
			return err
		}
	}
	return nil
}

func RegularOutputFileCount(outputDir string) (int, error) {
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, entry := range entries {
		if entry.Type().IsRegular() {
			count++
		}
	}
	return count, nil
}
