# Server API: Conversation History Storage

## Overview

Server API specification for storing conversation history sent from Claude Code hooks.
Each prompt-response pair is sent as a unit, with sequential integer IDs for easy reference.

## Core Concept

### Simple Conversation Storage

- Each interaction (prompt-response) sends only the new conversation entries
- Entries are **filtered** to contain only essential text (user/assistant messages)
- Server assigns sequential integer IDs to each entry
- Interactions reference conversation range via `conversation_start_id` and `conversation_end_id`
- Previous context is tracked via code change history (snapshot chain)

```
Session abc:
  Conversation ID 1-2:  Interaction 1 (user + assistant)
  Conversation ID 3-4:  Interaction 2 (user + assistant)
  ...
```

---

## Entry Filtering (Client-side)

Before sending, the client filters transcript entries:

| Entry Type | Action |
|------------|--------|
| `user` | Extract text strings from `message.content` array |
| `assistant` | Extract `text` field from items where `type='text'` |
| Other types | **Ignored** (tool_use, tool_result, etc.) |

**Filtered format:**
```json
{
  "entry_type": "user" | "assistant",
  "entry_data": "extracted text content"
}
```

---

## Endpoints

### 1. `POST /api/conversations`

Stores conversation entries on the server.

#### Request

**Headers:**
```
Content-Type: application/json
X-API-Key: {api_key}
```

**Body:**
```json
{
  "project_hash": "sha256_project_hash",
  "session_id": "claude_session_id",
  "entries": [
    {
      "entry_type": "user",
      "entry_data": "How do I implement a binary search?"
    },
    {
      "entry_type": "assistant",
      "entry_data": "Here's how to implement binary search in Go..."
    }
  ]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `project_hash` | string | O | Project identification hash |
| `session_id` | string | O | Claude Code session ID |
| `entries` | array | O | Filtered conversation entries |
| `entries[].entry_type` | string | O | `"user"` or `"assistant"` |
| `entries[].entry_data` | string | O | Text content |

#### Response

**Success (200 OK):**
```json
{
  "success": true,
  "entries_stored": 2,
  "start_id": 1,
  "end_id": 2
}
```

| Field | Type | Description |
|-------|------|-------------|
| `success` | boolean | Processing success status |
| `entries_stored` | number | Number of stored entries |
| `start_id` | number | First assigned conversation ID |
| `end_id` | number | Last assigned conversation ID |

---

### 2. `POST /api/snapshots` (Existing API)

Creates a snapshot of the project state.

#### Request Body

```json
{
  "project_hash": "...",
  "message": "...",
  "changes": [...],
  "claude_session_id": "...",
  "parent_snapshot_id": "..."
}
```

---

### 3. `POST /api/interactions` (Existing API - Extended)

Creates an interaction record (prompt-response pair) with conversation reference.

#### Request Body

```json
{
  "project_hash": "...",
  "message": "...",
  "changes": [...],
  "parent_snapshot_id": "...",
  "claude_session_id": "...",
  "started_at": "...",
  "ended_at": "...",
  "conversation_start_id": 1,
  "conversation_end_id": 2
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `conversation_start_id` | number | X | First conversation entry ID for this interaction |
| `conversation_end_id` | number | X | Last conversation entry ID for this interaction |

---

## Database Schema (Recommended)

### conversations table

```sql
CREATE TABLE conversations (
    id SERIAL PRIMARY KEY,  -- Sequential integer ID
    project_id INTEGER NOT NULL REFERENCES projects(id),
    session_id VARCHAR(255) NOT NULL,
    entry_type VARCHAR(20) NOT NULL,  -- 'user' or 'assistant'
    entry_data TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_conversations_project_session ON conversations(project_id, session_id);
```

### interactions table extension

```sql
ALTER TABLE interactions ADD COLUMN conversation_start_id INTEGER REFERENCES conversations(id);
ALTER TABLE interactions ADD COLUMN conversation_end_id INTEGER REFERENCES conversations(id);
```

### Query conversation for an interaction

```sql
SELECT c.* FROM conversations c
WHERE c.id BETWEEN
    (SELECT conversation_start_id FROM interactions WHERE id = ?)
    AND
    (SELECT conversation_end_id FROM interactions WHERE id = ?)
ORDER BY c.id;
```

---

## Client Behavior Summary

1. **user_prompt_submit**: Record current transcript line count
2. **stop**: Read entries from recorded line count to current end
3. **Filter**: Extract only user/assistant text content
4. **Send conversations**: POST filtered entries to `/api/conversations`, receive `start_id` and `end_id`
5. **Create interaction**: POST to `/api/interactions` with `conversation_start_id` and `conversation_end_id`

---

## Test

```bash
# Send conversation entries
curl -X POST http://localhost:5000/api/conversations \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your_api_key" \
  -d '{
    "project_hash": "test_project_hash",
    "session_id": "test_session_123",
    "entries": [
      {"entry_type": "user", "entry_data": "Hello, how are you?"},
      {"entry_type": "assistant", "entry_data": "I am doing well, thank you!"}
    ]
  }'

# Response: {"success": true, "entries_stored": 2, "start_id": 1, "end_id": 2}

# Create interaction with conversation reference
curl -X POST http://localhost:5000/api/interactions \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your_api_key" \
  -d '{
    "project_hash": "test_project_hash",
    "message": "[AUTO-POST] Hello",
    "changes": [],
    "parent_snapshot_id": "123",
    "claude_session_id": "test_session_123",
    "started_at": "2024-01-15T10:30:00Z",
    "ended_at": "2024-01-15T10:30:10Z",
    "conversation_start_id": 1,
    "conversation_end_id": 2
  }'
```
