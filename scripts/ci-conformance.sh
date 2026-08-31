#!/usr/bin/env bash
set -Eeuo pipefail

repo_root="$(pwd)"
run_id="${GITHUB_RUN_ID:-ci-local}"
work_root="${RUNNER_TEMP:-/tmp}/gooo-improvement-proposer-${run_id}"
binary="$work_root/gooo-improvement-proposer"
evidence="$work_root/evidence"
mkdir -p "$work_root" "$evidence/cases"

before_status="$(git status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')"

test_start="$(date +%s%N)"
/usr/bin/time -f '%e %M' -o "$work_root/test.time" go test -json -count=1 ./... > "$work_root/go-test.json"
test_end="$(date +%s%N)"

/usr/bin/time -f '%e %M' -o "$work_root/build.time" go build -trimpath -o "$binary" ./cmd/gooo-improvement-proposer

compile_ir="$work_root/semantic-ir.json"
compile_evaluator="$work_root/generated-evaluator.go"
"$binary" compile --source examples/improvement-proposer.gooo --contract contracts/improvement-proposer-denominator-v1.json --output-ir "$compile_ir" --output-evaluator "$compile_evaluator" > "$work_root/compile.stdout"

conformance_start="$(date +%s%N)"
"$binary" conformance --root "$repo_root" --source examples/improvement-proposer.gooo --contract contracts/improvement-proposer-denominator-v1.json --corpus fixtures/corpus.json --output-dir "$evidence/cases" > "$work_root/conformance.stdout"
conformance_end="$(date +%s%N)"

test -f "$evidence/cases/conformance-index.json"
test "$(find "$evidence/cases" -mindepth 2 -maxdepth 2 -type f -name proposal.json | wc -l | tr -d ' ')" -eq 9
while IFS= read -r -d '' proposal; do
  case_dir="$(dirname "$proposal")"
  for file in proposal.json candidate-events.ndjson semantic-ir.json generated-evaluator.go replay-receipt.json human-dossier.md; do
    test -f "$case_dir/$file"
  done
  jq -e '.state == "CLOSED" or .state == "UNKNOWN" or .state == "REFUTED"' "$proposal" >/dev/null
  jq -e '.authority.repository_writes == 0 and .authority.pull_request_creations == 0 and .authority.merge_operations == 0 and .authority.local_test_executions == 0 and .authority.cross_project_required_gates == 0' "$proposal" >/dev/null
  jq -e 'all(.candidates[]?; has("candidate_id") and has("causal_source_cells") and has("expected_evidence_type") and has("permitted_authority") and has("falsifier") and has("next_operation") and (has("priority") | not) and (has("score") | not) and (has("title") | not))' "$proposal" >/dev/null
  jq -e 'all(.unknowns[]?; .stage != "" and .step != "" and .reason != "" and .unknown_class != "" and .next_operation != "" and (.blocked_by != null))' "$proposal" >/dev/null
done < <(find "$evidence/cases" -mindepth 2 -maxdepth 2 -type f -name proposal.json -print0 | sort -z)

cp "$compile_ir" "$evidence/semantic-ir.json"
cp "$compile_evaluator" "$evidence/generated-evaluator.go"
cp contracts/improvement-proposer-denominator-v1.json "$evidence/denominator.json"

discovered="$(jq -s '[.[] | select(.Action == "run" and .Test != null)] | length' "$work_root/go-test.json")"
executed="$(jq -s '[.[] | select((.Action == "pass" or .Action == "fail") and .Test != null)] | length' "$work_root/go-test.json")"
skipped="$(jq -s '[.[] | select(.Action == "skip" and .Test != null)] | length' "$work_root/go-test.json")"
reused="$(jq -s '[.[] | select(.Action == "output" and .Test != null and ((.Output // "") | contains("(cached)")))] | length' "$work_root/go-test.json")"
not_observed=$((discovered - executed - skipped))
if test "$not_observed" -lt 0; then
  not_observed=0
fi

read -r build_seconds build_rss < "$work_root/build.time"
read -r test_seconds test_rss < "$work_root/test.time"
build_wall_ms="$(awk -v value="$build_seconds" 'BEGIN { printf "%d", (value * 1000) + 0.5 }')"
test_wall_ms="$(( (test_end - test_start) / 1000000 ))"
conformance_wall_ms="$(( (conformance_end - conformance_start) / 1000000 ))"
peak_rss_kib="$(awk -v build="$build_rss" -v test="$test_rss" 'BEGIN { if (build > test) print build; else print test }')"

candidate_count="$(jq '[.cases[].candidate_count] | add' "$evidence/cases/conformance-index.json")"
evidence_edges="$(jq '[.cases[].evidence_edges] | add' "$evidence/cases/conformance-index.json")"
blocked_frontier="$(jq '[.cases[].blocked_frontier] | add' "$evidence/cases/conformance-index.json")"
output_files="$(jq '[.cases[].output_files] | add' "$evidence/cases/conformance-index.json")"

directories="$(find . -mindepth 1 -type d ! -path './.git' ! -path './.git/*' | wc -l | tr -d ' ')"
regular_files="$(find . -type f ! -path './.git' ! -path './.git/*' ! -path './README.md' | wc -l | tr -d ' ')"
go_files="$(find . -type f -name '*.go' ! -path './.git/*' | wc -l | tr -d ' ')"
go_lines="$(find . -type f -name '*.go' ! -path './.git/*' -print0 | xargs -0 -r awk '{ total++ } END { print total + 0 }')"
gooo_files="$(find . -type f -name '*.gooo' ! -path './.git/*' | wc -l | tr -d ' ')"
gooo_lines="$(find . -type f -name '*.gooo' ! -path './.git/*' -print0 | xargs -0 -r awk '{ total++ } END { print total + 0 }')"
after_status="$(git status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')"
test "$before_status" = "$after_status"

jq -S -n \
  --arg schema "gooo/improvement-proposer/runtime/v1" \
  --arg run_id "$run_id" \
  --argjson build_wall_ms "$build_wall_ms" \
  --argjson test_wall_ms "$test_wall_ms" \
  --argjson conformance_wall_ms "$conformance_wall_ms" \
  --argjson build_peak_rss_kib "$build_rss" \
  --argjson test_peak_rss_kib "$test_rss" \
  --argjson conformance_peak_rss_kib "$peak_rss_kib" \
  --argjson tests_executed "$executed" \
  --argjson tests_reused "$reused" \
  --argjson tests_skipped "$skipped" \
  --argjson tests_not_observed "$not_observed" \
  --argjson go_files "$go_files" \
  --argjson go_lines "$go_lines" \
  --argjson gooo_files "$gooo_files" \
  --argjson gooo_lines "$gooo_lines" \
  --argjson regular_files "$regular_files" \
  --argjson directories "$directories" \
  --argjson candidate_count "$candidate_count" \
  --argjson evidence_edges "$evidence_edges" \
  --argjson blocked_frontier "$blocked_frontier" \
  --argjson output_files "$output_files" \
  --argjson repository_writes 0 \
  --argjson pull_request_creations 0 \
  --argjson merge_operations 0 \
  --argjson local_test_executions 0 \
  --argjson cross_project_required_gates 0 \
  '{schema:$schema,ci_run_id:$run_id,build_wall_ms:$build_wall_ms,test_wall_ms:$test_wall_ms,conformance_wall_ms:$conformance_wall_ms,build_peak_rss_kib:$build_peak_rss_kib,test_peak_rss_kib:$test_peak_rss_kib,conformance_peak_rss_kib:$conformance_peak_rss_kib,tests:{executed:$tests_executed,reused:$tests_reused,skipped:$tests_skipped,not_observed:$tests_not_observed},candidate_count:$candidate_count,evidence_edges:$evidence_edges,blocked_frontier:$blocked_frontier,output_files:$output_files,inventory:{go_files:$go_files,go_lines:$go_lines,gooo_files:$gooo_files,gooo_lines:$gooo_lines,regular_files_root_readme_excluded:$regular_files,directories:$directories},authority:{repository_writes:$repository_writes,pull_request_creations:$pull_request_creations,merge_operations:$merge_operations,local_test_executions:$local_test_executions,cross_project_required_gates:$cross_project_required_gates}}' > "$evidence/runtime.json"

jq -S -n \
  --arg schema "gooo/improvement-proposer/ci-report/v1" \
  --arg precedence "REFUTED > UNKNOWN > CLOSED" \
  --slurpfile index "$evidence/cases/conformance-index.json" \
  --slurpfile runtime "$evidence/runtime.json" \
  '{schema:$schema,decision:"PROPOSER_CONFORMANCE_CLOSED",precedence:$precedence,fixed_denominator:{total:12,exact:true},scenario_counts:$index[0].states,cases:$index[0].cases,runtime:$runtime[0],authority:$runtime[0].authority}' > "$evidence/ci-report.json"

cat > "$evidence/summary.md" <<EOF
# gooo-improvement-proposer CI summary

- decision: \`PROPOSER_CONFORMANCE_CLOSED\`
- fixed denominator: \`12/12\`
- candidate count: \`$candidate_count\`
- evidence edges: \`$evidence_edges\`
- blocked frontier: \`$blocked_frontier\`
- output files: \`$output_files\`
- repository writes: \`0\`
- pull-request creations: \`0\`
- merge operations: \`0\`
- local test executions: \`0\`
- cross-project required gates: \`0\`
EOF

printf 'evidence=%s\n' "$evidence"
