package proposer

import "encoding/json"

const (
	ProtocolSchema = "gooo/improvement-proposer/protocol/v1"
	SourceSchema   = "gooo/improvement-proposer/v1"
	IRSchema       = "gooo/improvement-proposer/ir/v1"

	StateClosed  State = "CLOSED"
	StateUnknown State = "UNKNOWN"
	StateRefuted State = "REFUTED"

	EventProposed   = "PROPOSED"
	EventBlocked    = "BLOCKED"
	EventNoRequired = "NO_CANDIDATE_REQUIRED"
)

var StatePrecedence = []State{StateRefuted, StateUnknown, StateClosed}

var FixedOutputFiles = []string{
	"proposal.json",
	"candidate-events.ndjson",
	"semantic-ir.json",
	"generated-evaluator.go",
	"replay-receipt.json",
	"human-dossier.md",
}

type State string

type ProofChoice string

type LedgerRelease struct {
	Repository  string `json:"repository"`
	Tag         string `json:"tag"`
	CommitSHA   string `json:"commit_sha"`
	Asset       string `json:"asset"`
	AssetDigest string `json:"asset_digest"`
	LedgerDigest string `json:"ledger_digest"`
	Immutable   bool   `json:"immutable"`
}

type LedgerBinding struct {
	Release             LedgerRelease `json:"release"`
	ExpectedTag         string        `json:"expected_tag"`
	ExpectedCommitSHA   string        `json:"expected_commit_sha"`
	ExpectedLedgerDigest string       `json:"expected_ledger_digest"`
}

type CapabilityRecord struct {
	RecordID       string   `json:"record_id"`
	CapabilityID   string   `json:"capability_id"`
	State          State    `json:"state"`
	SourceCell     string   `json:"source_cell"`
	EvidenceDigest string   `json:"evidence_digest"`
	RecordDigest   string   `json:"record_digest"`
	Reason         string   `json:"reason"`
	BlockedBy      []string `json:"blocked_by"`
}

type CounterexampleRecord struct {
	RecordID       string   `json:"record_id"`
	CapabilityID   string   `json:"capability_id"`
	State          State    `json:"state"`
	SourceCell     string   `json:"source_cell"`
	EvidenceDigest string   `json:"evidence_digest"`
	RecordDigest   string   `json:"record_digest"`
	Falsifier      string   `json:"falsifier"`
	BlockedBy      []string `json:"blocked_by"`
}

type UtilityVector struct {
	MemoryKib         int `json:"memory_kib"`
	BuildWallMs       int `json:"build_wall_ms"`
	TestWallMs        int `json:"test_wall_ms"`
	ConformanceWallMs int `json:"conformance_wall_ms"`
}

type UtilityRecord struct {
	RecordID       string         `json:"record_id"`
	CapabilityID   string         `json:"capability_id"`
	State          State          `json:"state"`
	SourceCell     string         `json:"source_cell"`
	EvidenceDigest string         `json:"evidence_digest"`
	RecordDigest   string         `json:"record_digest"`
	Before         *UtilityVector `json:"before"`
	After          *UtilityVector `json:"after"`
}

type Input struct {
	Schema          string                 `json:"schema"`
	CaseID          string                 `json:"case_id"`
	Ledger          LedgerBinding         `json:"ledger"`
	Capabilities    []CapabilityRecord    `json:"capabilities"`
	Counterexamples []CounterexampleRecord `json:"counterexamples"`
	Utilities       []UtilityRecord       `json:"utilities"`
}

type Contract struct {
	Schema             string         `json:"schema"`
	DenominatorID      string         `json:"denominator_id"`
	Total              int            `json:"total"`
	ProofChoices       []string       `json:"proof_choices"`
	IndicatorClasses   []string       `json:"indicator_classes"`
	Cells              []ContractCell `json:"cells"`
	StatePrecedence    []State         `json:"state_precedence"`
	MaxCandidates      int            `json:"max_candidates"`
	UnknownFields      []string       `json:"unknown_fields"`
	NoScores           bool           `json:"no_scores"`
	NoPercentages      bool           `json:"no_percentages"`
	NoNaturalPriority  bool           `json:"no_natural_language_priority"`
	AuthorityZeros     bool           `json:"authority_zeros"`
}

type ContractCell struct {
	Ordinal          int    `json:"ordinal"`
	ID               string `json:"id"`
	Activity         string `json:"activity"`
	ClaimID          string `json:"claim_id"`
	OperationID      string `json:"operation_id"`
	Stage            string `json:"stage"`
	Step             string `json:"step"`
	ProofChoice      string `json:"proof_choice"`
	IndicatorClass   string `json:"indicator_class"`
	MetricID         string `json:"metric_id"`
	MetricPath       string `json:"metric_path"`
	GeneratedArtifact string `json:"generated_artifact"`
	Evaluator        string `json:"evaluator"`
}

type SourceActivity struct {
	ID             string `json:"id"`
	Activity      string `json:"activity"`
	ClaimID       string `json:"claim_id"`
	OperationID   string `json:"operation_id"`
	Stage         string `json:"stage"`
	Step          string `json:"step"`
	ProofChoice   string `json:"proof_choice"`
	IndicatorClass string `json:"indicator_class"`
	SourceLine    int    `json:"source_line"`
}

type SemanticIR struct {
	Schema         string           `json:"schema"`
	SourcePath     string           `json:"source_path"`
	SourceDigest   string           `json:"source_digest"`
	ContractDigest string           `json:"contract_digest"`
	Total          int              `json:"total"`
	Activities     []SourceActivity `json:"activities"`
}

type UnknownDetail struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

type Candidate struct {
	CandidateID       string   `json:"candidate_id"`
	CapabilityID      string   `json:"capability_id"`
	CausalSourceCells []string `json:"causal_source_cells"`
	ExpectedEvidence  string   `json:"expected_evidence_type"`
	PermittedAuthority []string `json:"permitted_authority"`
	Falsifier         string   `json:"falsifier"`
	NextOperation     string   `json:"next_operation"`
	BlockedBy         []string `json:"blocked_by"`
}

type CandidateEvent struct {
	Schema             string   `json:"schema"`
	Ordinal            int      `json:"ordinal"`
	EventID            string   `json:"event_id"`
	EventType          string   `json:"event_type"`
	InputDigest        string   `json:"input_digest"`
	CandidateID        string   `json:"candidate_id"`
	CapabilityID       string   `json:"capability_id"`
	CausalSourceCells  []string `json:"causal_source_cells"`
	ExpectedEvidence   string   `json:"expected_evidence_type"`
	PermittedAuthority []string `json:"permitted_authority"`
	Falsifier          string   `json:"falsifier"`
	NextOperation      string   `json:"next_operation"`
	BlockedBy          []string `json:"blocked_by"`
	Reason             string   `json:"reason"`
	PreviousEventDigest string  `json:"previous_event_digest"`
	EventDigest        string   `json:"event_digest"`
}

type ArtifactBinding struct {
	Artifact string `json:"artifact"`
	Digest   string `json:"digest"`
}

type Proposal struct {
	Schema              string            `json:"schema"`
	CaseID              string            `json:"case_id"`
	InputDigest         string            `json:"input_digest"`
	LedgerRelease       LedgerRelease     `json:"ledger_release"`
	State               State             `json:"state"`
	Decision            string            `json:"decision"`
	StatePrecedence     []State           `json:"state_precedence"`
	CandidateCount      int               `json:"candidate_count"`
	EvidenceEdges       int               `json:"evidence_edges"`
	BlockedFrontier     int               `json:"blocked_frontier"`
	Candidates          []Candidate       `json:"candidates"`
	Unknowns            []UnknownDetail   `json:"unknowns"`
	Refutations         []string          `json:"refutations"`
	Bindings            []ArtifactBinding `json:"bindings"`
	Authority           Authority         `json:"authority"`
}

type Authority struct {
	RepositoryWrites          int `json:"repository_writes"`
	PullRequestCreations      int `json:"pull_request_creations"`
	MergeOperations           int `json:"merge_operations"`
	LocalTestExecutions       int `json:"local_test_executions"`
	CrossProjectRequiredGates int `json:"cross_project_required_gates"`
}

type ReplayReceipt struct {
	Schema             string            `json:"schema"`
	CaseID             string            `json:"case_id"`
	InputDigest        string            `json:"input_digest"`
	Deterministic      bool              `json:"deterministic"`
	CandidateIDs       []string          `json:"candidate_ids"`
	CandidateCount     int               `json:"candidate_count"`
	EvidenceEdges      int               `json:"evidence_edges"`
	BlockedFrontier    int               `json:"blocked_frontier"`
	CandidateEventsDigest string          `json:"candidate_events_digest"`
	ProposalDigest     string            `json:"proposal_digest"`
	OutputFiles        []ArtifactBinding `json:"output_files"`
}

type RuntimeMetrics struct {
	BuildWallMs               int `json:"build_wall_ms"`
	TestWallMs                int `json:"test_wall_ms"`
	ConformanceWallMs         int `json:"conformance_wall_ms"`
	BuildPeakRSSKib            int `json:"build_peak_rss_kib"`
	TestPeakRSSKib             int `json:"test_peak_rss_kib"`
	ConformancePeakRSSKib      int `json:"conformance_peak_rss_kib"`
	TestsExecuted              int `json:"tests_executed"`
	TestsReused                int `json:"tests_reused"`
	TestsSkipped               int `json:"tests_skipped"`
	TestsNotObserved           int `json:"tests_not_observed"`
	GoFiles                    int `json:"go_files"`
	GoLines                    int `json:"go_lines"`
	GoooFiles                  int `json:"gooo_files"`
	GoooLines                  int `json:"gooo_lines"`
	RegularFilesRootReadmeExcluded int `json:"regular_files_root_readme_excluded"`
	Directories                int `json:"directories"`
	CandidateCount             int `json:"candidate_count"`
	EvidenceEdges              int `json:"evidence_edges"`
	BlockedFrontier            int `json:"blocked_frontier"`
	OutputFiles                int `json:"output_files"`
	RepositoryWrites           int `json:"repository_writes"`
	PullRequestCreations       int `json:"pull_request_creations"`
	MergeOperations            int `json:"merge_operations"`
	LocalTestExecutions        int `json:"local_test_executions"`
	CrossProjectRequiredGates  int `json:"cross_project_required_gates"`
}

type Evaluation struct {
	Proposal      Proposal
	Events        []CandidateEvent
	SemanticIRRaw []byte
	GeneratedRaw  []byte
	Replay        ReplayReceipt
	Dossier       string
}

func (r RuntimeMetrics) MarshalJSON() ([]byte, error) {
	type alias RuntimeMetrics
	return json.Marshal(alias(r))
}
