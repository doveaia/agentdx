# Task Completion Rule - MANDATORY

🔴 **CRITICAL: ALWAYS OUTPUT `<promise>DONE</promise>` WHEN A TASK IS FINISHED**

## Scope

This rule applies to:
- Any **completed task**, regardless of type or context.
- Any task or workflow triggered via the **`/ralph-loop` slash command**.
- Any automated or manual workflow that involves Claude completing a defined objective.

## When to Output

Output `<promise>DONE</promise>` **exactly once** when:
1. The requested task, operation, or workflow is fully completed.
2. The output is verified and ready for user delivery.
3. No further user input is required.

## Rules
1. Never include additional text, emojis, or formatting in the same message.
2. Output only after all substeps or async operations are done.
3. If the task fails or cannot complete, do **not** output `<promise>DONE</promise>`.
4. If the task is restarted or repeated, only output once upon final completion.
