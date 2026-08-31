package proposer

import "testing"

func TestUtilityWithoutExactPairBecomesUnknownAndProposed(t *testing.T) {
	input := baseInput()
	input.Utilities = []UtilityRecord{{
		RecordID: "utility-1", CapabilityID: "capability-1", State: StateClosed,
		SourceCell: "cell-05", EvidenceDigest: digest("3"), RecordDigest: digest("4"),
	}}
	evaluation, err := Propose(input, digest("5"), testIR(), []byte("ir"), []byte("evaluator"), digest("6"), digest("7"))
	if err != nil {
		t.Fatal(err)
	}
	if evaluation.Proposal.State != StateUnknown || evaluation.Proposal.CandidateCount != 1 {
		t.Fatalf("state=%s candidates=%d", evaluation.Proposal.State, evaluation.Proposal.CandidateCount)
	}
	if len(evaluation.Proposal.Unknowns) != 1 || ValidateUnknown(evaluation.Proposal.Unknowns[0]) != nil {
		t.Fatalf("unknown coordinates were not preserved: %#v", evaluation.Proposal.Unknowns)
	}
	if evaluation.Proposal.Candidates[0].ExpectedEvidence != "EXACT_BEFORE_AFTER_UTILITY_RECEIPT" {
		t.Fatalf("unexpected evidence type %q", evaluation.Proposal.Candidates[0].ExpectedEvidence)
	}
}

func TestKnownContradictionHasPrecedenceAndNoCandidate(t *testing.T) {
	input := baseInput()
	input.Capabilities = append(input.Capabilities, CapabilityRecord{
		RecordID: "capability-duplicate", CapabilityID: "capability-1", State: StateUnknown,
		SourceCell: "cell-03", EvidenceDigest: digest("d"), RecordDigest: digest("e"), Reason: "conflicting record",
	})
	evaluation, err := Propose(input, digest("8"), testIR(), []byte("ir"), []byte("evaluator"), digest("6"), digest("7"))
	if err != nil {
		t.Fatal(err)
	}
	if evaluation.Proposal.State != StateRefuted || evaluation.Proposal.CandidateCount != 0 {
		t.Fatalf("state=%s candidates=%d", evaluation.Proposal.State, evaluation.Proposal.CandidateCount)
	}
	if len(evaluation.Proposal.Refutations) == 0 {
		t.Fatal("known contradiction was not retained")
	}
}

func TestClosedInputEmitsNoExperimentCandidate(t *testing.T) {
	input := baseInput()
	input.Utilities = []UtilityRecord{{
		RecordID: "utility-1", CapabilityID: "capability-1", State: StateClosed,
		SourceCell: "cell-05", EvidenceDigest: digest("3"), RecordDigest: digest("4"),
		Before: &UtilityVector{MemoryKib: 10, BuildWallMs: 20, TestWallMs: 30, ConformanceWallMs: 40},
		After:  &UtilityVector{MemoryKib: 9, BuildWallMs: 19, TestWallMs: 29, ConformanceWallMs: 39},
	}}
	evaluation, err := Propose(input, digest("9"), testIR(), []byte("ir"), []byte("evaluator"), digest("6"), digest("7"))
	if err != nil {
		t.Fatal(err)
	}
	if evaluation.Proposal.State != StateClosed || evaluation.Proposal.CandidateCount != 0 || evaluation.Proposal.Decision != "NO_EXPERIMENT_REQUIRED" {
		t.Fatalf("unexpected closed proposal: %#v", evaluation.Proposal)
	}
}

func baseInput() Input {
	return Input{
		Schema: ProtocolSchema + "/input/v1", CaseID: "test",
		Ledger: LedgerBinding{
			Release: LedgerRelease{
				Repository: "kimjooyoon/gooo-self-improvement-ledger", Tag: "v0.11.0",
				CommitSHA: "e7e408f6631a6b54effa9a9da9bb33947b35cb09", Asset: "ledger.zip",
				AssetDigest: digest("1"), LedgerDigest: digest("2"), Immutable: true,
			},
			ExpectedTag: "v0.11.0", ExpectedCommitSHA: "e7e408f6631a6b54effa9a9da9bb33947b35cb09", ExpectedLedgerDigest: digest("2"),
		},
		Capabilities: []CapabilityRecord{{
			RecordID: "capability-record-1", CapabilityID: "capability-1", State: StateClosed,
			SourceCell: "cell-03", EvidenceDigest: digest("a"), RecordDigest: digest("b"), Reason: "closed",
		}},
	}
}

func testIR() SemanticIR {
	activities := make([]SourceActivity, 12)
	for i := range activities {
		activities[i] = SourceActivity{ID: "cell", Activity: "activity", ClaimID: "claim", OperationID: "operation", Stage: "stage", Step: "step", ProofChoice: "FOUNDATION", IndicatorClass: "DRIVER", SourceLine: i + 1}
	}
	return SemanticIR{Schema: IRSchema, SourceDigest: digest("source"), ContractDigest: digest("contract"), Total: 12, Activities: activities}
}

func digest(value string) string {
	result := ""
	for len(result) < 64 {
		result += value
	}
	return "sha256:" + result[:64]
}
