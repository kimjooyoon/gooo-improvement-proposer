package proposer

import (
	"fmt"
	"sort"
	"strings"
)

const (
	unknownLedgerStale       = "STALE_LEDGER_RELEASE"
	unknownMissingUtility    = "EXACT_BEFORE_AFTER_UTILITY_MISSING"
	unknownCapabilityDirect  = "CAPABILITY_EVIDENCE_DIRECT_MISSING"
	unknownCapabilityBlocked = "CAPABILITY_EVIDENCE_DEPENDENCY_BLOCKED"
	unknownCounterexample    = "COUNTEREXAMPLE_REPLAY_EVIDENCE_MISSING"
	refutedContradiction     = "KNOWN_EVIDENCE_CONTRADICTION"
	refutedLedgerDigest      = "LEDGER_RELEASE_DIGEST_INVALID"
	refutedAuthority         = "PROPOSER_AUTHORITY_BOUNDARY_VIOLATED"
)

func ValidateInput(input Input) error {
	if input.Schema != ProtocolSchema+"/input/v1" || input.CaseID == "" {
		return fmt.Errorf("INVALID_INPUT_HEADER")
	}
	if err := validateLedgerSyntax(input.Ledger.Release); err != nil {
		return err
	}
	if input.Ledger.ExpectedTag == "" || input.Ledger.ExpectedCommitSHA == "" || input.Ledger.ExpectedLedgerDigest == "" {
		return fmt.Errorf("INCOMPLETE_LEDGER_EXPECTATION")
	}
	if err := validateCommit(input.Ledger.ExpectedCommitSHA); err != nil {
		return err
	}
	if err := ValidateDigest(input.Ledger.ExpectedLedgerDigest); err != nil {
		return err
	}
	for _, record := range input.Capabilities {
		if record.RecordID == "" || record.CapabilityID == "" || record.SourceCell == "" || !validState(record.State) || !validDigestFields(record.EvidenceDigest, record.RecordDigest) {
			return fmt.Errorf("INVALID_CAPABILITY_RECORD_%s", record.RecordID)
		}
		if record.State == StateUnknown && record.Reason == "" {
			return fmt.Errorf("UNKNOWN_CAPABILITY_REASON_MISSING_%s", record.RecordID)
		}
	}
	for _, record := range input.Counterexamples {
		if record.RecordID == "" || record.CapabilityID == "" || record.SourceCell == "" || !validState(record.State) || !validDigestFields(record.EvidenceDigest, record.RecordDigest) || record.Falsifier == "" {
			return fmt.Errorf("INVALID_COUNTEREXAMPLE_RECORD_%s", record.RecordID)
		}
	}
	for _, record := range input.Utilities {
		if record.RecordID == "" || record.CapabilityID == "" || record.SourceCell == "" || !validState(record.State) || !validDigestFields(record.EvidenceDigest, record.RecordDigest) {
			return fmt.Errorf("INVALID_UTILITY_RECORD_%s", record.RecordID)
		}
		if err := validateUtilityVector(record.Before); err != nil {
			return fmt.Errorf("utility %s before: %w", record.RecordID, err)
		}
		if err := validateUtilityVector(record.After); err != nil {
			return fmt.Errorf("utility %s after: %w", record.RecordID, err)
		}
	}
	return nil
}

func validateLedgerSyntax(release LedgerRelease) error {
	if release.Repository == "" || release.Tag == "" || release.Asset == "" || release.CommitSHA == "" || !release.Immutable {
		return fmt.Errorf("%s", refutedLedgerDigest)
	}
	if err := validateCommit(release.CommitSHA); err != nil {
		return err
	}
	if err := ValidateDigest(release.AssetDigest); err != nil {
		return err
	}
	if err := ValidateDigest(release.LedgerDigest); err != nil {
		return err
	}
	return nil
}

func validDigestFields(values ...string) bool {
	for _, value := range values {
		if ValidateDigest(value) != nil {
			return false
		}
	}
	return true
}

func validateUtilityVector(vector *UtilityVector) error {
	if vector == nil {
		return nil
	}
	if vector.MemoryKib < 0 || vector.BuildWallMs < 0 || vector.TestWallMs < 0 || vector.ConformanceWallMs < 0 {
		return fmt.Errorf("NEGATIVE_UTILITY_VALUE")
	}
	return nil
}

func validState(value State) bool {
	return value == StateClosed || value == StateUnknown || value == StateRefuted
}

func ValidateUnknown(value UnknownDetail) error {
	if value.Stage == "" || value.Step == "" || value.Reason == "" || value.UnknownClass == "" || value.NextOperation == "" {
		return fmt.Errorf("unknown detail does not contain the six required fields")
	}
	return nil
}

func Propose(input Input, inputDigest string, ir SemanticIR, irRaw, generatedRaw []byte, contractDigest, evaluatorDigest string) (Evaluation, error) {
	if err := ValidateInput(input); err != nil {
		return Evaluation{}, err
	}
	if ir.Total != 12 || len(ir.Activities) != 12 {
		return Evaluation{}, fmt.Errorf("semantic activity denominator is not 12")
	}
	issues, refutations, stale := deriveInputIssues(input)
	candidates := make([]Candidate, 0, 12)
	if !stale && !hasKnownContradiction(refutations) {
		candidates = deriveCandidates(input)
	}
	if len(candidates) > 12 {
		return Evaluation{}, fmt.Errorf("%s", refutedAuthority)
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].CandidateID < candidates[j].CandidateID })
	blockedFrontier := countBlockedFrontier(candidates, issues)
	state := StateClosed
	decision := "NO_EXPERIMENT_REQUIRED"
	if len(refutations) > 0 {
		state = StateRefuted
		decision = "PROPOSAL_BLOCKED_BY_REFUTATION"
	} else if len(issues) > 0 {
		state = StateUnknown
		if len(candidates) > 0 {
			decision = "PROPOSE_NEXT_EXPERIMENT"
		} else {
			decision = "PROPOSAL_BLOCKED_BY_UNKNOWN"
		}
	} else if len(candidates) > 0 {
		decision = "PROPOSE_NEXT_EXPERIMENT"
	}
	if len(refutations) > 0 && hasKnownContradiction(refutations) {
		candidates = nil
		blockedFrontier = countBlockedFrontier(candidates, issues)
	}
	events := buildEvents(inputDigest, candidates, issues, refutations)
	eventsRaw := eventsNDJSON(events)
	proposal := Proposal{
		Schema: ProtocolSchema + "/proposal/v1", CaseID: input.CaseID, InputDigest: inputDigest,
		LedgerRelease: input.Ledger.Release, State: state, Decision: decision,
		StatePrecedence: append([]State(nil), StatePrecedence...), CandidateCount: len(candidates),
		EvidenceEdges: evidenceEdgeCount(candidates), BlockedFrontier: blockedFrontier,
		Candidates: candidates, Unknowns: issues, Refutations: refutations,
		Bindings: []ArtifactBinding{
			{Artifact: "semantic-ir.json", Digest: DigestBytes(irRaw)},
			{Artifact: "generated-evaluator.go", Digest: DigestBytes(generatedRaw)},
			{Artifact: "candidate-events.ndjson", Digest: DigestBytes(eventsRaw)},
			{Artifact: "evaluator", Digest: evaluatorDigest},
		},
		Authority: Authority{},
	}
	proposalRaw, err := jsonMarshalIndent(proposal)
	if err != nil {
		return Evaluation{}, err
	}
	proposalDigest := DigestBytes(proposalRaw)
	replay := ReplayReceipt{
		Schema: ProtocolSchema + "/replay-receipt/v1", CaseID: input.CaseID, InputDigest: inputDigest,
		Deterministic: true, CandidateIDs: candidateIDs(candidates), CandidateCount: len(candidates),
		EvidenceEdges: evidenceEdgeCount(candidates), BlockedFrontier: blockedFrontier,
		CandidateEventsDigest: DigestBytes(eventsRaw), ProposalDigest: proposalDigest,
		OutputFiles: outputBindings(proposalRaw, eventsRaw, irRaw, generatedRaw),
	}
	dossier := RenderDossier(proposal, replay, events)
	return Evaluation{Proposal: proposal, Events: events, SemanticIRRaw: irRaw, GeneratedRaw: generatedRaw, Replay: replay, Dossier: dossier}, nil
}

func deriveInputIssues(input Input) ([]UnknownDetail, []string, bool) {
	issues := make([]UnknownDetail, 0)
	refutations := make([]string, 0)
	stale := false
	if !input.Ledger.Release.Immutable || input.Ledger.Release.Tag != input.Ledger.ExpectedTag || input.Ledger.Release.CommitSHA != input.Ledger.ExpectedCommitSHA || input.Ledger.Release.LedgerDigest != input.Ledger.ExpectedLedgerDigest {
		stale = true
		issues = append(issues, UnknownDetail{
			Stage: "LEDGER_BINDING", Step: "VERIFY_CURRENT_IMMUTABLE_RELEASE", Reason: unknownLedgerStale,
			UnknownClass: "STALE_INPUT", NextOperation: "REFRESH_LEDGER_RELEASE_AND_DIGEST", BlockedBy: []string{"ledger-release"},
		})
	}
	capabilityEvidence := map[string]string{}
	for _, record := range input.Capabilities {
		if previous, exists := capabilityEvidence[record.CapabilityID]; exists && previous != record.EvidenceDigest {
			refutations = append(refutations, refutedContradiction+":"+record.CapabilityID)
		}
		capabilityEvidence[record.CapabilityID] = record.EvidenceDigest
		switch record.State {
		case StateUnknown:
			class := "DIRECT_MISSING"
			blocked := append([]string(nil), record.BlockedBy...)
			if len(blocked) > 0 {
				class = "DEPENDENCY_BLOCKED"
			}
			issues = append(issues, UnknownDetail{
				Stage: "CAPABILITY_RECORD", Step: "REQUIRE_EXACT_CAPABILITY_EVIDENCE", Reason: record.Reason,
				UnknownClass: class, NextOperation: "COLLECT_EXACT_CAPABILITY_EVIDENCE", BlockedBy: sortedStrings(blocked),
			})
		case StateRefuted:
			refutations = append(refutations, record.RecordID+":"+record.Reason)
		}
	}
	counterexampleEvidence := map[string]string{}
	for _, record := range input.Counterexamples {
		if previous, exists := counterexampleEvidence[record.RecordID]; exists && previous != record.EvidenceDigest {
			refutations = append(refutations, refutedContradiction+":"+record.RecordID)
		}
		counterexampleEvidence[record.RecordID] = record.EvidenceDigest
		switch record.State {
		case StateUnknown:
			issues = append(issues, UnknownDetail{
				Stage: "COUNTEREXAMPLE_RECORD", Step: "REPLAY_IMMUTABLE_COUNTEREXAMPLE", Reason: unknownCounterexample,
				UnknownClass: "DIRECT_MISSING", NextOperation: "REPLAY_COUNTEREXAMPLE_WITH_EXACT_EVIDENCE", BlockedBy: sortedStrings(record.BlockedBy),
			})
		case StateRefuted:
			refutations = append(refutations, record.RecordID+":"+record.Falsifier)
		}
	}
	utilityEvidence := map[string]string{}
	for _, record := range input.Utilities {
		if previous, exists := utilityEvidence[record.RecordID]; exists && previous != record.EvidenceDigest {
			refutations = append(refutations, refutedContradiction+":"+record.RecordID)
		}
		utilityEvidence[record.RecordID] = record.EvidenceDigest
		exact := record.Before != nil && record.After != nil
		if record.State == StateRefuted {
			refutations = append(refutations, record.RecordID+":UTILITY_EVIDENCE_REFUTED")
			continue
		}
		if !exact || record.State == StateUnknown {
			issues = append(issues, UnknownDetail{
				Stage: "UTILITY_RECORD", Step: "REQUIRE_EXACT_BEFORE_AFTER_PAIR", Reason: unknownMissingUtility,
				UnknownClass: "DIRECT_MISSING", NextOperation: "COLLECT_EXACT_BEFORE_AFTER_UTILITY_PAIR", BlockedBy: nil,
			})
		}
	}
	sort.Slice(issues, func(i, j int) bool { return issueKey(issues[i]) < issueKey(issues[j]) })
	sort.Strings(refutations)
	return issues, refutations, stale
}

func deriveCandidates(input Input) []Candidate {
	candidates := make([]Candidate, 0)
	for _, record := range input.Capabilities {
		if record.State == StateClosed {
			continue
		}
		candidates = append(candidates, Candidate{
			CandidateID: "capability/" + record.CapabilityID, CapabilityID: record.CapabilityID,
			CausalSourceCells: []string{record.SourceCell}, ExpectedEvidence: "EXACT_CAPABILITY_CONFORMANCE_RECEIPT",
			PermittedAuthority: permittedAuthority(), Falsifier: "CAPABILITY_RECORD_NOT_CLOSED_WITH_EXACT_EVIDENCE",
			NextOperation: "COLLECT_EXACT_CAPABILITY_EVIDENCE", BlockedBy: sortedStrings(record.BlockedBy),
		})
	}
	for _, record := range input.Counterexamples {
		if record.State == StateClosed {
			continue
		}
		candidates = append(candidates, Candidate{
			CandidateID: "counterexample/" + record.RecordID, CapabilityID: record.CapabilityID,
			CausalSourceCells: []string{record.SourceCell}, ExpectedEvidence: "EXACT_COUNTEREXAMPLE_REPLAY_RECEIPT",
			PermittedAuthority: permittedAuthority(), Falsifier: record.Falsifier,
			NextOperation: "REPLAY_COUNTEREXAMPLE_WITH_EXACT_EVIDENCE", BlockedBy: sortedStrings(record.BlockedBy),
		})
	}
	for _, record := range input.Utilities {
		if record.State == StateClosed && record.Before != nil && record.After != nil {
			continue
		}
		candidates = append(candidates, Candidate{
			CandidateID: "utility/" + record.RecordID, CapabilityID: record.CapabilityID,
			CausalSourceCells: []string{record.SourceCell}, ExpectedEvidence: "EXACT_BEFORE_AFTER_UTILITY_RECEIPT",
			PermittedAuthority: permittedAuthority(), Falsifier: "UTILITY_BEFORE_AFTER_PAIR_MISSING_OR_NON_EXACT",
			NextOperation: "COLLECT_EXACT_BEFORE_AFTER_UTILITY_PAIR", BlockedBy: nil,
		})
	}
	return candidates
}

func permittedAuthority() []string {
	return []string{"READ_IMMUTABLE_LEDGER", "EXECUTE_DECLARED_EVALUATOR", "WRITE_CALLER_OWNED_TEMP_OUTPUT"}
}

func buildEvents(inputDigest string, candidates []Candidate, unknowns []UnknownDetail, refutations []string) []CandidateEvent {
	events := make([]CandidateEvent, 0, len(candidates)+1)
	previous := ""
	for index, candidate := range candidates {
		event := CandidateEvent{
			Schema: ProtocolSchema + "/candidate-event/v1", Ordinal: index + 1,
			EventID: fmt.Sprintf("candidate-event-%03d", index+1), EventType: EventProposed, InputDigest: inputDigest,
			CandidateID: candidate.CandidateID, CapabilityID: candidate.CapabilityID,
			CausalSourceCells: append([]string(nil), candidate.CausalSourceCells...), ExpectedEvidence: candidate.ExpectedEvidence,
			PermittedAuthority: append([]string(nil), candidate.PermittedAuthority...), Falsifier: candidate.Falsifier,
			NextOperation: candidate.NextOperation, BlockedBy: append([]string(nil), candidate.BlockedBy...),
			Reason: "DERIVED_FROM_EXPLICIT_EVIDENCE_GAP", PreviousEventDigest: previous,
		}
		event.EventDigest = DigestEvent(event)
		previous = event.EventDigest
		events = append(events, event)
	}
	if len(events) == 0 {
		reason := "NO_UNMET_EVIDENCE"
		eventType := EventNoRequired
		if len(refutations) > 0 {
			reason = strings.Join(refutations, ";")
			eventType = EventBlocked
		} else if len(unknowns) > 0 {
			reason = unknowns[0].Reason
			eventType = EventBlocked
		}
		event := CandidateEvent{
			Schema: ProtocolSchema + "/candidate-event/v1", Ordinal: 1, EventID: "proposal-event-001", EventType: eventType,
			InputDigest: inputDigest, Reason: reason,
		}
		event.EventDigest = DigestEvent(event)
		events = append(events, event)
	}
	return events
}

func outputBindings(proposalRaw, eventsRaw, irRaw, generatedRaw []byte) []ArtifactBinding {
	return []ArtifactBinding{
		{Artifact: "proposal.json", Digest: DigestBytes(proposalRaw)},
		{Artifact: "candidate-events.ndjson", Digest: DigestBytes(eventsRaw)},
		{Artifact: "semantic-ir.json", Digest: DigestBytes(irRaw)},
		{Artifact: "generated-evaluator.go", Digest: DigestBytes(generatedRaw)},
		{Artifact: "replay-receipt.json", Digest: "deferred-to-avoid-cycle"},
		{Artifact: "human-dossier.md", Digest: "deferred-to-render"},
	}
}

func eventsNDJSON(events []CandidateEvent) []byte {
	var builder strings.Builder
	for _, event := range events {
		raw, _ := jsonMarshal(event)
		builder.Write(raw)
		builder.WriteByte('\n')
	}
	return []byte(builder.String())
}

func candidateIDs(candidates []Candidate) []string {
	ids := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		ids = append(ids, candidate.CandidateID)
	}
	return ids
}

func evidenceEdgeCount(candidates []Candidate) int {
	count := 0
	for _, candidate := range candidates {
		count += len(candidate.CausalSourceCells)
	}
	return count
}

func countBlockedFrontier(candidates []Candidate, unknowns []UnknownDetail) int {
	unique := map[string]bool{}
	for _, candidate := range candidates {
		for _, blocker := range candidate.BlockedBy {
			unique[blocker] = true
		}
	}
	for _, unknown := range unknowns {
		for _, blocker := range unknown.BlockedBy {
			unique[blocker] = true
		}
	}
	return len(unique)
}

func sortedStrings(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

func issueKey(value UnknownDetail) string {
	return value.Stage + "|" + value.Step + "|" + value.Reason + "|" + strings.Join(value.BlockedBy, ",")
}

func hasKnownContradiction(refutations []string) bool {
	for _, value := range refutations {
		if strings.Contains(value, refutedContradiction) {
			return true
		}
	}
	return false
}
