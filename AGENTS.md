# AGENTS.md

## Workflow

Every task — feature, bugfix, refactor — follows this loop. No exceptions.

### 1. Analyze
Run 2+ parallel agents to explore the existing implementation and identify gaps. If scope is large, add more agents to cover all paths and edge cases.

### 2. Plan
Synthesize findings. Run a plan-review agent before implementation. Break large scopes into small, independently shippable tasks.

### 3. Develop
One agent per task. Small scope per agent. Launch parallel agents for independent tasks.

### 4. Code Review
Immediately after each dev agent completes, run a separate review agent on that task's output. Fix findings with separate dev agents.

### 5. Cross-Cutting Review
After all tasks land, run 5 parallel review agents across the full codebase. Fix findings with targeted dev agents. Repeat until all agents return CLEAR.

### Rules
- **Separate agents for dev and review.** Never review your own code.
- **Small scope per agent.** If an agent's task is too big, split it.
- **Parallelize.** Independent tasks and independent reviews run in parallel.
- **Fix with separate agents.** After a review finds bugs, launch targeted fix agents — don't edit inline.
- **Verify after every change.** `go vet ./... && go test -race ./...` after every fix.
- **No shortcuts.** Full scope, every feature, no deferrals.
