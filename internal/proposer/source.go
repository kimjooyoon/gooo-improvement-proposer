package proposer

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
)

func LoadContract(path string) (Contract, []byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Contract{}, nil, err
	}
	var contract Contract
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&contract); err != nil {
		return Contract{}, nil, err
	}
	if err := ValidateContract(contract); err != nil {
		return Contract{}, nil, err
	}
	return contract, raw, nil
}

func ValidateContract(contract Contract) error {
	if contract.Schema != ProtocolSchema+"/denominator/v1" || contract.DenominatorID == "" {
		return errors.New("INVALID_CONTRACT_HEADER")
	}
	if contract.Total != 12 || len(contract.Cells) != 12 || contract.MaxCandidates != 12 {
		return errors.New("INVALID_FIXED_DENOMINATOR")
	}
	if len(contract.ProofChoices) != 3 || len(contract.IndicatorClasses) != 3 {
		return errors.New("INVALID_CATEGORY_DENOMINATOR")
	}
	if len(contract.StatePrecedence) != 3 || contract.StatePrecedence[0] != StateRefuted || contract.StatePrecedence[1] != StateUnknown || contract.StatePrecedence[2] != StateClosed {
		return errors.New("INVALID_STATE_PRECEDENCE")
	}
	if len(contract.UnknownFields) != 6 {
		return errors.New("INVALID_UNKNOWN_FIELD_DENOMINATOR")
	}
	for _, required := range []string{"stage", "step", "reason", "unknown_class", "next_operation", "blocked_by"} {
		found := slices.Contains(contract.UnknownFields, required)
		if !found {
			return fmt.Errorf("MISSING_UNKNOWN_FIELD_%s", required)
		}
	}
	if !contract.NoScores || !contract.NoPercentages || !contract.NoNaturalPriority || !contract.AuthorityZeros {
		return errors.New("FORBIDDEN_INFERENCE_OR_AUTHORITY_ENABLED")
	}
	seenOrdinals := map[int]bool{}
	seenActivities := map[string]bool{}
	proofCounts := map[string]int{}
	indicatorCounts := map[string]int{}
	for index, cell := range contract.Cells {
		if cell.Ordinal != index+1 || seenOrdinals[cell.Ordinal] || cell.ID == "" || cell.Activity == "" || cell.ClaimID == "" || cell.OperationID == "" || cell.Stage == "" || cell.Step == "" || cell.MetricID == "" || cell.MetricPath == "" || cell.GeneratedArtifact == "" || cell.Evaluator == "" {
			return fmt.Errorf("INVALID_CONTRACT_CELL_%d", index+1)
		}
		if seenActivities[cell.Activity] || !validProof(cell.ProofChoice) || !validIndicator(cell.IndicatorClass) {
			return fmt.Errorf("INVALID_CONTRACT_ACTIVITY_%s", cell.Activity)
		}
		seenOrdinals[cell.Ordinal] = true
		seenActivities[cell.Activity] = true
		proofCounts[cell.ProofChoice]++
		indicatorCounts[cell.IndicatorClass]++
	}
	for _, choice := range contract.ProofChoices {
		if proofCounts[choice] != 4 {
			return fmt.Errorf("PROOF_CHOICE_NOT_BALANCED_%s", choice)
		}
	}
	for _, class := range contract.IndicatorClasses {
		if indicatorCounts[class] != 4 {
			return fmt.Errorf("INDICATOR_CLASS_NOT_BALANCED_%s", class)
		}
	}
	return nil
}

func CompileSource(sourcePath string, source []byte, contract Contract, contractDigest string) (SemanticIR, error) {
	if err := ValidateContract(contract); err != nil {
		return SemanticIR{}, err
	}
	lines := strings.Split(string(source), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != `@gooo schema="gooo/improvement-proposer/v1"` {
		return SemanticIR{}, errors.New("GOOO_SOURCE_SCHEMA_HEADER_MISSING")
	}
	activities := make([]SourceActivity, 0, len(contract.Cells))
	for lineNumber, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if !strings.HasPrefix(line, "activity ") {
			continue
		}
		attrs, err := parseAttributes(strings.TrimPrefix(line, "activity "))
		if err != nil {
			return SemanticIR{}, fmt.Errorf("source line %d: %w", lineNumber+1, err)
		}
		activity := SourceActivity{
			ID: attrs["id"], Activity: attrs["activity"], ClaimID: attrs["claim_id"], OperationID: attrs["operation_id"],
			Stage: attrs["stage"], Step: attrs["step"], ProofChoice: attrs["proof_choice"], IndicatorClass: attrs["indicator_class"], SourceLine: lineNumber + 1,
		}
		if activity.ID == "" || activity.Activity == "" || activity.ClaimID == "" || activity.OperationID == "" || activity.Stage == "" || activity.Step == "" || !validProof(activity.ProofChoice) || !validIndicator(activity.IndicatorClass) {
			return SemanticIR{}, fmt.Errorf("source line %d: INCOMPLETE_ACTIVITY_METADATA", lineNumber+1)
		}
		activities = append(activities, activity)
	}
	if len(activities) != 12 {
		return SemanticIR{}, errors.New("GOOO_SOURCE_ACTIVITY_COUNT_NOT_12")
	}
	for index, cell := range contract.Cells {
		activity := activities[index]
		if activity.ID != cell.ID || activity.Activity != cell.Activity || activity.ClaimID != cell.ClaimID || activity.OperationID != cell.OperationID || activity.Stage != cell.Stage || activity.Step != cell.Step || activity.ProofChoice != cell.ProofChoice || activity.IndicatorClass != cell.IndicatorClass {
			return SemanticIR{}, fmt.Errorf("GOOO_ACTIVITY_BINDING_MISMATCH_%d", cell.Ordinal)
		}
	}
	return SemanticIR{
		Schema: IRSchema, SourcePath: sourcePath, SourceDigest: DigestBytes(source), ContractDigest: contractDigest,
		Total: len(activities), Activities: activities,
	}, nil
}

func SemanticIRBytes(ir SemanticIR) ([]byte, error) {
	raw, err := json.MarshalIndent(ir, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

func GenerateEvaluator(ir SemanticIR, irDigest, contractDigest string) []byte {
	var builder strings.Builder
	builder.WriteString("// Code generated by gooo. DO NOT EDIT.\n\n")
	builder.WriteString("package generated\n\n")
	fmt.Fprintf(&builder, "const SourceDigest = %q\n", ir.SourceDigest)
	fmt.Fprintf(&builder, "const SemanticIRDigest = %q\n", irDigest)
	fmt.Fprintf(&builder, "const ContractDigest = %q\n", contractDigest)
	fmt.Fprintf(&builder, "const MetaActivityCount = %d\n\n", ir.Total)
	builder.WriteString("type Activity struct {\n\tID string\n\tActivity string\n\tClaimID string\n\tOperationID string\n\tStage string\n\tStep string\n\tProofChoice string\n\tIndicatorClass string\n}\n\n")
	builder.WriteString("var Activities = []Activity{\n")
	for _, activity := range ir.Activities {
		fmt.Fprintf(&builder, "\t{ID: %q, Activity: %q, ClaimID: %q, OperationID: %q, Stage: %q, Step: %q, ProofChoice: %q, IndicatorClass: %q},\n", activity.ID, activity.Activity, activity.ClaimID, activity.OperationID, activity.Stage, activity.Step, activity.ProofChoice, activity.IndicatorClass)
	}
	builder.WriteString("}\n")
	return []byte(builder.String())
}

func parseAttributes(input string) (map[string]string, error) {
	attrs := map[string]string{}
	for position := 0; position < len(input); {
		for position < len(input) && input[position] == ' ' {
			position++
		}
		if position == len(input) {
			break
		}
		keyStart := position
		for position < len(input) && input[position] != '=' && input[position] != ' ' {
			position++
		}
		if keyStart == position || position >= len(input) || input[position] != '=' {
			return nil, errors.New("INVALID_ATTRIBUTE")
		}
		key := input[keyStart:position]
		if _, exists := attrs[key]; exists {
			return nil, errors.New("DUPLICATE_ATTRIBUTE")
		}
		position++
		if position >= len(input) || input[position] != '"' {
			return nil, errors.New("ATTRIBUTES_MUST_BE_QUOTED")
		}
		valueStart := position
		position++
		for position < len(input) {
			if input[position] == '"' && input[position-1] != '\\' {
				position++
				break
			}
			position++
		}
		if position > len(input) || position == 0 || input[position-1] != '"' {
			return nil, errors.New("UNTERMINATED_ATTRIBUTE")
		}
		value, err := strconv.Unquote(input[valueStart:position])
		if err != nil {
			return nil, err
		}
		attrs[key] = value
	}
	return attrs, nil
}

func validProof(value string) bool {
	return value == "FOUNDATION" || value == "COHERENCE" || value == "REGRESSION"
}

func validIndicator(value string) bool {
	return value == "DRIVER" || value == "OUTCOME" || value == "GUARDRAIL"
}
