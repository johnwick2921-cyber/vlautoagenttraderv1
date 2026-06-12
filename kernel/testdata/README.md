# Golden test data

This directory holds snapshot outputs of the futures-mode AI prompts.

The tests in `kernel/engine_prompt_golden_test.go` compare current
`BuildFuturesSystemPrompt` / `BuildFuturesUserPrompt` output against
these files; any change to the prompt template fails the test.

## Refreshing goldens after an intentional template change

    go test ./kernel/ -run TestBuildFutures -update

Commit the updated files alongside the template change. This makes
prompt-template diffs auditable in code review (the .txt files appear
in the diff).
