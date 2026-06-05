# Tool Calling Implementation Plan

This document outlines the plan to implement tool calling in lemon-chat, starting with a "Get Current Time" tool.

## Overview
Tool calling allows the LLM to request the execution of specific functions to retrieve real-time data or perform actions. The backend will intercept these requests, execute the corresponding logic, and feed the result back into the conversation context.

## 1. Backend Changes

### Database Schema (`internal/store/characters.go`)
- Update the `character` table to store which tools are enabled for that character.
- **Implementation:** Add a `tools` column (text/json) to the `character` table to store a list of enabled tool IDs.
- **Migration:** Create a new migration in `internal/store/store.go` to add this column.

### Character API (`internal/server/characters.go`)
- Update the `Create` and `Update` handlers to accept and save the `tools` configuration.

### Model Integration (`internal/server/messages.go`)
- **Tool Definition:** Create a registry of available tools. The "Get Current Time" tool will be defined here with its name and description.
- **Request Modification:** When sending a request to the LLM, include the tool definitions if the selected character has them enabled.
- **Response Handling:**
    - Check if the LLM response contains a tool call.
    - If a tool call is detected:
        1. Execute the tool logic (e.g., `time.Now()` for the time tool).
        2. Append the tool call and the tool result to the conversation history in the database.
        3. (Optional) Re-call the LLM with the tool result to generate a final natural language response for the user.
- **Context Management:** Ensure tool calls and results are correctly formatted as `tool` role messages (if supported by the model provider) or as system/user messages to maintain context.

### Tool Logic
- Implement a `ToolExecutor` that maps tool IDs to Go functions.
- `getTime`: Returns the current server time formatted as a string.

## 2. Frontend Changes

### Character Editor (`static/js/settings-character-edit.js` & `static/settings/character-edit.html`)
- Add a "Tools" section to the character edit page.
- Add a checkbox for "Get Current Time".
- Update the save logic to send the enabled tools list to the backend.

## 3. Workflow Summary
1. User sends message $\rightarrow$ Server identifies character $\rightarrow$ Server checks enabled tools.
2. Server sends prompt + tool definitions to LLM.
3. LLM returns a tool call request.
4. Server executes tool $\rightarrow$ Saves tool call & result to DB.
5. Server sends result back to LLM $\rightarrow$ LLM returns final answer.
6. Final answer is streamed to user.

## Future Extensibility
- The system will be designed so that adding new tools only requires adding a definition to the registry and a corresponding function in the `ToolExecutor`.
