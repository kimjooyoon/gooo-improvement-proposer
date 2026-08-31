package proposer

import (
	"fmt"
	"strings"
)

func RenderDossier(proposal Proposal, replay ReplayReceipt, events []CandidateEvent) string {
	var builder strings.Builder
	fmt.Fprintln(&builder, "# Gooo Improvement Proposer Human Dossier")
	fmt.Fprintln(&builder)
	fmt.Fprintf(&builder, "case: `%s`\n", proposal.CaseID)
	fmt.Fprintf(&builder, "state: `%s`\n", proposal.State)
	fmt.Fprintf(&builder, "decision: `%s`\n", proposal.Decision)
	fmt.Fprintf(&builder, "precedence: `%s`\n", joinStates(proposal.StatePrecedence))
	fmt.Fprintln(&builder, "priority method: `none; candidates are ordered by canonical identifier only`")
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, "## Exact proposal counts")
	fmt.Fprintln(&builder)
	fmt.Fprintf(&builder, "- candidate count: `%d`\n", proposal.CandidateCount)
	fmt.Fprintf(&builder, "- evidence edges: `%d`\n", proposal.EvidenceEdges)
	fmt.Fprintf(&builder, "- blocked frontier: `%d`\n", proposal.BlockedFrontier)
	fmt.Fprintf(&builder, "- output files: `%d`\n", len(FixedOutputFiles))
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, "## Candidates")
	fmt.Fprintln(&builder)
	if len(proposal.Candidates) == 0 {
		fmt.Fprintln(&builder, "No experiment candidate was emitted.")
	} else {
		fmt.Fprintln(&builder, "| candidate | capability | source cells | evidence | authority | falsifier | next operation | blocked by |")
		fmt.Fprintln(&builder, "|---|---|---|---|---|---|---|---|")
		for _, candidate := range proposal.Candidates {
			fmt.Fprintf(&builder, "| %s | %s | %s | %s | %s | %s | %s | %s |\n", candidate.CandidateID, candidate.CapabilityID, strings.Join(candidate.CausalSourceCells, ","), candidate.ExpectedEvidence, strings.Join(candidate.PermittedAuthority, ","), candidate.Falsifier, candidate.NextOperation, strings.Join(candidate.BlockedBy, ","))
		}
	}
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, "## Unknown coordinates")
	fmt.Fprintln(&builder)
	if len(proposal.Unknowns) == 0 {
		fmt.Fprintln(&builder, "None.")
	} else {
		for _, unknown := range proposal.Unknowns {
			fmt.Fprintf(&builder, "- stage=`%s`, step=`%s`, reason=`%s`, unknown_class=`%s`, next_operation=`%s`, blocked_by=`%s`\n", unknown.Stage, unknown.Step, unknown.Reason, unknown.UnknownClass, unknown.NextOperation, strings.Join(unknown.BlockedBy, ","))
		}
	}
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, "## Replay")
	fmt.Fprintln(&builder)
	fmt.Fprintf(&builder, "- deterministic: `%t`\n", replay.Deterministic)
	fmt.Fprintf(&builder, "- candidate ids: `%s`\n", strings.Join(replay.CandidateIDs, ","))
	fmt.Fprintf(&builder, "- event records: `%d`\n", len(events))
	fmt.Fprintf(&builder, "- candidate-events digest: `%s`\n", replay.CandidateEventsDigest)
	fmt.Fprintf(&builder, "- proposal digest: `%s`\n", replay.ProposalDigest)
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, "## Authority boundary")
	fmt.Fprintln(&builder)
	fmt.Fprintln(&builder, "The evaluator reads immutable evidence and writes only the caller-owned output directory. Repository writes, pull-request creation, merge operations, local test executions, and required cross-project gates are all exact integer zero.")
	return builder.String()
}

func joinStates(values []State) string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		items = append(items, string(value))
	}
	return strings.Join(items, " > ")
}
