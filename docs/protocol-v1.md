# gooo improvement proposer protocol v1

## Scope

The proposer has one semantic responsibility: convert exact, immutable
evidence gaps into a bounded set of experiment declarations. It does not
select among candidates, schedule their operations, edit source, create a pull
request, merge a branch, or treat missing evidence as success.

The ledger release is bound by repository, tag, commit, asset, asset digest,
ledger digest, and `immutable=true`. The input also carries the expected tag,
commit, and ledger digest. A mismatch is a stale-input `UNKNOWN` with a
refresh operation and a named blocker.

## Evidence records

Each capability, counterexample, and utility record has an explicit source
cell, evidence digest, and record digest. Capability and counterexample states
are `CLOSED`, `UNKNOWN`, or `REFUTED`. Utility records additionally carry exact
`before` and `after` vectors:

```json
{
  "memory_kib": 100,
  "build_wall_ms": 200,
  "test_wall_ms": 300,
  "conformance_wall_ms": 400
}
```

Both vectors are required for utility closure. There is no delta score or
weighted objective. A missing pair always remains `UNKNOWN`.

## Candidate rule

Candidates are derived in canonical identifier order from:

| evidence gap | candidate identifier | expected evidence | next operation |
| --- | --- | --- | --- |
| capability not CLOSED | `capability/<capability_id>` | exact capability conformance receipt | `COLLECT_EXACT_CAPABILITY_EVIDENCE` |
| counterexample not CLOSED | `counterexample/<record_id>` | exact counterexample replay receipt | `REPLAY_COUNTEREXAMPLE_WITH_EXACT_EVIDENCE` |
| utility pair absent or utility not CLOSED | `utility/<record_id>` | exact before/after utility receipt | `COLLECT_EXACT_BEFORE_AFTER_UTILITY_PAIR` |

The candidate set is bounded at 12. Exceeding that bound is a fail-closed
authority error. A known contradiction is preserved as `REFUTED` and prevents
candidate derivation. A refuted individual record remains visible as a
candidate cause, while its proposal state retains `REFUTED` precedence.

The causal edge count is the exact sum of candidate source-cell counts. The
blocked frontier is the exact count of distinct named blockers. These are
counts, not rankings.

## UNKNOWN coordinates

Every UNKNOWN detail contains all six operational fields:

```text
stage, step, reason, unknown_class, next_operation, blocked_by
```

`DIRECT_MISSING` may have an empty `blocked_by`. Dependency, stale-input, and
other blocked evidence names the minimal known blocker set. A known
contradiction is never lowered to UNKNOWN.

## Output and authority

One invocation writes only the six fixed artifacts listed in the root README.
`proposal.json` carries the state, counts, candidates, unknown coordinates,
refutations, input digest, ledger release identity, artifact bindings, and
zero-valued authority fields. `candidate-events.ndjson` is an append-only
digest chain. `replay-receipt.json` binds the input, candidate identifiers,
counts, event digest, proposal digest, and fixed artifact names. The dossier is
human-readable and repeats exact counts and the authority boundary.

All paths are written through temporary files and renames under the caller's
existing absolute output directory. Input files are never opened for write.
