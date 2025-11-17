AWS_REGION=eu-west-2 ANTHROPIC_MODEL=eu.anthropic.claude-sonnet-4-5-20250929-v1:0 CLAUDE_CODE_USE_BEDROCK=1 AWS_PROFILE=ai-admin claude --dangerously-skip-permissions --append-system-prompt "$(cat SYSTEM_PROMPT.md )" $@
claude --dangerously-skip-permissions --append-system-prompt "$(cat SYSTEM_PROMPT.md )" $@
