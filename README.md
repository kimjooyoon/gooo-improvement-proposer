# gooo-improvement-proposer

`gooo-improvement-proposer` derives the next bounded Gooo self-improvement
experiment from an immutable self-improvement ledger and exact capability,
counterexample, and utility records. It grants semantic proposal authority
that the neighboring repositories do not provide:

- `gooo-improvement-frontier` schedules graph operations and identifies
  parallel or serialized work.
- `gooo-improvement-selector` compares already-declared candidates.
- `gooo-self-improvement-ledger` records capability states and evidence.
- this repository derives candidate declarations from UNKNOWN, REFUTED, or
  missing utility evidence.

The proposer is read-only with respect to its input repository. Its only write
target is a caller-owned temporary output directory. Repository writes,
pull-request creation, merge operations, local test executions, and required
cross-project gates are fixed integer zero. It does not produce a title, score,
priority, percentage, or language-model estimate.

## Evidence chain

```text
.gooo source -> semantic IR -> generated evaluator -> proposal.json
                                      |                |
                                      v                v
                              candidate-events.ndjson  replay-receipt.json
                                                       |
                                                       v
                                               human-dossier.md
```

Every emitted candidate contains its causal source cells, expected evidence
type, permitted authority, falsifier, and next operation. Candidate order is
the canonical candidate identifier only; it is not a ranking.

Utility evidence is closed only when both `before` and `after` are present as
exact non-negative integer vectors. If the pair is absent, the result is
`UNKNOWN` and the next experiment is an exact utility-pair collection.

## Fixed contract

The denominator is exactly 12 `.gooo` activities, bound one-to-one to the
contract cells and generated evaluator. Proof choices are balanced:

```text
FOUNDATION 4   COHERENCE 4   REGRESSION 4
DRIVER 4       OUTCOME 4     GUARDRAIL 4
```

State precedence is `REFUTED > UNKNOWN > CLOSED`. A known contradiction stops
candidate derivation. A stale or non-current ledger is `UNKNOWN` until its
release identity and digest are refreshed. A normal all-closed input can
produce zero candidates.

The controlled conformance corpus includes normal zero-candidate and empty
frontier cases, deterministic replay, a minimal blocked frontier, a stale
ledger, missing exact utility evidence, a known contradiction, and explicit
refuted records. Normal, UNKNOWN, and REFUTED each have at least three cases.

## CI-only verification

GitHub Actions is the verification authority and uses Go 1.27. The workflow
runs format, `go fix`, build, test, vet, compiler, deterministic replay, and
conformance gates. Local Go build, test, vet, formatting, fix, compiler, and
conformance execution is intentionally outside this repository's process.

Run the CLI in a caller-owned absolute temporary directory:

```text
gooo-improvement-proposer propose \
  --input fixtures/cases/unknown-missing-utility.json \
  --output-dir /tmp/gooo-improvement-proposer-out
```

The `propose` command writes exactly:

```text
proposal.json
candidate-events.ndjson
semantic-ir.json
generated-evaluator.go
replay-receipt.json
human-dossier.md
```

See [docs/protocol-v1.md](docs/protocol-v1.md) for the input and output
contract. Release provenance is recorded in
[docs/release-v0.1.0.md](docs/release-v0.1.0.md).

## License

MIT.
