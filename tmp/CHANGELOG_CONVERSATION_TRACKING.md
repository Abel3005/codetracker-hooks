# 대화 히스토리 서버 저장 기능 구현

## 개요

Claude Code 훅을 통해 사용자와 Claude 간의 대화 히스토리를 자동으로 서버에 저장하는 기능을 추가했습니다. 이 기능은 기존 파일 변경 추적 기능과 함께 동작하며, AI 코딩 세션의 전체 맥락을 보존합니다.

## 변경된 파일 목록

| 파일 | 변경 유형 | 설명 |
|------|----------|------|
| `.claude/hooks/stop.js` | 수정 | 대화 전송 로직 추가 |
| `.claude/hooks/user_prompt_submit.js` | 수정 | 새 캐시 구조 지원 |
| `.codetracker/config.json` | 수정 | `conversation_tracking` 설정 추가 |

---

## 1. `.claude/hooks/stop.js` 변경 사항

### 1.1 새로 추가된 함수들

#### `getSnapshotFiles(snapshot)` (라인 53-64)

기존 캐시 파일 구조와 새 구조를 모두 지원하기 위한 헬퍼 함수입니다.

```javascript
/**
 * Get file data from snapshot (handles both old and new structure)
 * Old structure: { "path/file.js": { hash, size } }
 * New structure: { files: { "path/file.js": { hash, size } }, transcript: {...} }
 * @param {object} snapshot - Snapshot data from last_snapshot.json
 * @returns {object} File data map
 */
function getSnapshotFiles(snapshot) {
  if (!snapshot) return null;
  // New structure has 'files' field
  if (snapshot.files) return snapshot.files;
  // Old structure: snapshot itself is the file map (no 'files' or 'transcript' keys)
  const keys = Object.keys(snapshot);
  if (keys.length > 0 && !keys.includes('files') && !keys.includes('transcript')) {
    return snapshot;
  }
  return null;
}
```

**역할:**
- 이전 버전의 `last_snapshot.json` (플랫 구조)과 새 버전 (`files` 필드 포함)을 모두 처리
- 마이그레이션 없이 기존 캐시 파일과 호환

---

#### `getSnapshotTranscript(snapshot)` (라인 71-74)

캐시에서 대화 전송 상태를 추출하는 헬퍼 함수입니다.

```javascript
/**
 * Get transcript state from snapshot
 * @param {object} snapshot - Snapshot data from last_snapshot.json
 * @returns {object|null} Transcript state or null
 */
function getSnapshotTranscript(snapshot) {
  if (!snapshot) return null;
  return snapshot.transcript || null;
}
```

**역할:**
- `last_snapshot.json`에서 `transcript` 필드 추출
- 대화 전송 상태 (마지막 전송 라인 번호, 세션 ID 등) 관리

---

#### `readTranscriptEntries(transcriptPath, startLine, maxEntries)` (라인 84-122)

Claude Code의 대화 히스토리 JSONL 파일을 읽는 함수입니다.

```javascript
/**
 * Read new entries from transcript JSONL file
 * Handles rewind by detecting when saved line count exceeds actual file lines
 * @param {string} transcriptPath - Path to JSONL file
 * @param {number} startLine - Line number to start reading from (0-indexed)
 * @param {number} maxEntries - Maximum entries to read
 * @returns {Array} Array of parsed JSONL entries with line_index
 */
function readTranscriptEntries(transcriptPath, startLine = 0, maxEntries = 100) {
  if (!transcriptPath || !fs.existsSync(transcriptPath)) {
    return [];
  }

  try {
    const content = fs.readFileSync(transcriptPath, 'utf8');
    const lines = content.split('\n');
    const entries = [];

    // Filter non-empty lines and track their original indices
    const nonEmptyLines = [];
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i].trim();
      if (line) {
        nonEmptyLines.push({ index: i, content: line });
      }
    }

    // Rewind detection: if startLine >= total lines, reset to 0
    const effectiveStartLine = (startLine >= nonEmptyLines.length) ? 0 : startLine;

    for (let i = effectiveStartLine; i < nonEmptyLines.length && entries.length < maxEntries; i++) {
      const { index, content: lineContent } = nonEmptyLines[i];
      try {
        entries.push({
          line_index: index,
          data: JSON.parse(lineContent)
        });
      } catch (e) {
        // Skip malformed JSON lines
      }
    }

    return entries;
  } catch (e) {
    return [];
  }
}
```

**역할:**
- Claude Code가 생성하는 JSONL 형식의 transcript 파일 파싱
- 증분 읽기: `startLine`부터 `maxEntries`개만 읽어 성능 최적화
- **Rewind 감지**: 저장된 라인 번호가 실제 파일 라인 수보다 크면 처음부터 다시 읽기
- 빈 라인 필터링 및 원본 라인 인덱스 추적

**반환 형식:**
```javascript
[
  { line_index: 0, data: { type: "user", message: {...} } },
  { line_index: 1, data: { type: "assistant", message: {...} } },
  ...
]
```

---

#### `sendTranscriptEntries(entries, sessionId, credentials, config)` (라인 132-157)

서버로 대화 데이터를 전송하는 함수입니다.

```javascript
/**
 * Send transcript entries to server
 * @param {Array} entries - Array of transcript entries to send
 * @param {string} sessionId - Claude session ID
 * @param {object} credentials - User credentials with api_key
 * @param {object} config - Configuration with server_url
 * @returns {boolean} True if successful, false otherwise
 */
async function sendTranscriptEntries(entries, sessionId, credentials, config) {
  const serverUrl = config.server_url || 'http://localhost:5000';

  try {
    const response = await fetch(`${serverUrl}/api/conversations`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-API-Key': credentials.api_key
      },
      body: JSON.stringify({
        project_hash: credentials.current_project_hash,
        session_id: sessionId,
        entries: entries.map(e => ({
          line_index: e.line_index,
          ...e.data
        }))
      })
    });

    return response.ok;
  } catch (e) {
    // Silently fail - never block Claude operation
    return false;
  }
}
```

**역할:**
- `POST /api/conversations` 엔드포인트로 대화 엔트리 전송
- 네트워크 오류 시 조용히 실패 (Claude Code 동작에 영향 없음)

**요청 형식:**
```json
{
  "project_hash": "프로젝트_해시",
  "session_id": "Claude_세션_ID",
  "entries": [
    { "line_index": 0, "type": "user", "message": {...} },
    { "line_index": 1, "type": "assistant", "message": {...} }
  ]
}
```

---

### 1.2 `createPostPromptSnapshot()` 함수 변경 (라인 372-383)

**변경 전:**
```javascript
const snapshotFileData = {};
for (const [filePath, info] of Object.entries(currentFiles)) {
  snapshotFileData[filePath] = {
    hash: info.hash,
    size: info.size
  };
}
saveJSON(lastSnapshotFile, snapshotFileData);

return postSnapshotId;
```

**변경 후:**
```javascript
const filesData = {};
for (const [filePath, info] of Object.entries(currentFiles)) {
  filesData[filePath] = {
    hash: info.hash,
    size: info.size
  };
}
// Return files data instead of saving here - will be saved with transcript state in main()
return { postSnapshotId, filesData };
```

**변경 이유:**
- 파일 데이터와 대화 상태를 함께 저장하기 위해 `main()`에서 통합 저장
- 반환값을 `{ postSnapshotId, filesData }` 객체로 변경

---

### 1.3 `main()` 함수 변경 (라인 393-487)

**주요 추가 로직:**

#### 1) `transcript_path` 추출 (라인 403)
```javascript
const transcriptPath = hookData.transcript_path || '';
```

Claude Code가 stdin으로 전달하는 `transcript_path`를 추출합니다.

#### 2) 대화 전송 로직 (라인 423-462)
```javascript
// Send conversation entries if enabled and transcript path is available
if (result && transcriptPath && config && credentials &&
    config.conversation_tracking?.enabled !== false) {

  // Load previous transcript state
  const previousSnapshotData = loadJSON(lastSnapshotFile);
  const previousTranscript = getSnapshotTranscript(previousSnapshotData);

  // Determine start line (same session continues, different session starts fresh)
  const maxEntries = config.conversation_tracking?.max_entries_per_request || 100;
  let startLine = 0;
  if (previousTranscript && previousTranscript.session_id === sessionData.claude_session_id) {
    startLine = (previousTranscript.last_sent_line || -1) + 1;
  }

  // Read and send transcript entries
  const entries = readTranscriptEntries(transcriptPath, startLine, maxEntries);

  if (entries.length > 0) {
    const sent = await sendTranscriptEntries(
      entries,
      sessionData.claude_session_id,
      credentials,
      config
    );

    if (sent) {
      // Update transcript state with last sent line
      const lastEntry = entries[entries.length - 1];
      transcriptState = {
        session_id: sessionData.claude_session_id,
        last_sent_line: lastEntry.line_index,
        last_sent_at: new Date().toISOString()
      };
    }
  } else if (previousTranscript) {
    // No new entries but preserve previous state
    transcriptState = previousTranscript;
  }
}
```

**동작 흐름:**
1. `conversation_tracking.enabled`가 `false`가 아니면 실행
2. 이전 전송 상태 로드 (`last_snapshot.json`의 `transcript` 필드)
3. 같은 세션이면 마지막 전송 라인 다음부터, 새 세션이면 처음부터 읽기
4. 대화 엔트리 읽기 및 서버 전송
5. 전송 성공 시 새 전송 상태 저장

#### 3) 통합 캐시 저장 (라인 464-471)
```javascript
// Save snapshot state with files and transcript
if (result && result.filesData) {
  const snapshotData = {
    files: result.filesData,
    ...(transcriptState && { transcript: transcriptState })
  };
  saveJSON(lastSnapshotFile, snapshotData);
}
```

파일 해시 데이터와 대화 전송 상태를 하나의 JSON 파일로 저장합니다.

---

## 2. `.claude/hooks/user_prompt_submit.js` 변경 사항

### 2.1 새로 추가된 함수들

`stop.js`와 동일한 헬퍼 함수들이 추가되었습니다:
- `getSnapshotFiles(snapshot)` (라인 52-63)
- `getSnapshotTranscript(snapshot)` (라인 70-73)

### 2.2 `createPrePromptSnapshot()` 함수 변경

**변경 전:**
```javascript
const currentFiles = getTrackedFiles(config);
const previousSnapshot = loadJSON(lastSnapshotFile);
const changes = calculateDiff(currentFiles, previousSnapshot);
```

**변경 후:**
```javascript
const currentFiles = getTrackedFiles(config);
const previousSnapshotData = loadJSON(lastSnapshotFile);
const previousFiles = getSnapshotFiles(previousSnapshotData);
const previousTranscript = getSnapshotTranscript(previousSnapshotData);
const changes = calculateDiff(currentFiles, previousFiles);
```

### 2.3 캐시 저장 로직 변경

**변경 전:**
```javascript
const snapshotData = {};
for (const [filePath, info] of Object.entries(currentFiles)) {
  snapshotData[filePath] = {
    hash: info.hash,
    size: info.size
  };
}
saveJSON(lastSnapshotFile, snapshotData);
```

**변경 후:**
```javascript
const filesData = {};
for (const [filePath, info] of Object.entries(currentFiles)) {
  filesData[filePath] = {
    hash: info.hash,
    size: info.size
  };
}
const snapshotData = {
  files: filesData,
  // Preserve existing transcript state if any
  ...(previousTranscript && { transcript: previousTranscript })
};
saveJSON(lastSnapshotFile, snapshotData);
```

**변경 이유:**
- 새 캐시 구조 (`files` 필드) 사용
- `stop.js`에서 저장한 `transcript` 상태 보존

---

## 3. `.codetracker/config.json` 변경 사항

### 추가된 설정

```json
{
  "conversation_tracking": {
    "enabled": true,
    "max_entries_per_request": 100
  }
}
```

| 필드 | 타입 | 기본값 | 설명 |
|------|------|--------|------|
| `enabled` | boolean | `true` | 대화 추적 기능 활성화 여부 |
| `max_entries_per_request` | number | `100` | 한 번에 전송할 최대 엔트리 수 |

---

## 4. 캐시 파일 구조 변경

### `.codetracker/cache/last_snapshot.json`

**이전 구조 (v2):**
```json
{
  "src/index.js": { "hash": "sha256...", "size": 1234 },
  "src/utils.js": { "hash": "sha256...", "size": 567 }
}
```

**새 구조 (v3):**
```json
{
  "files": {
    "src/index.js": { "hash": "sha256...", "size": 1234 },
    "src/utils.js": { "hash": "sha256...", "size": 567 }
  },
  "transcript": {
    "session_id": "abc123-def456",
    "last_sent_line": 42,
    "last_sent_at": "2024-01-15T10:36:00.000Z"
  }
}
```

**호환성:**
- 양쪽 훅 모두 이전 구조와 새 구조를 자동으로 감지하고 처리
- 기존 캐시 파일이 있어도 마이그레이션 불필요

---

## 5. 서버 API 요구사항

### `POST /api/conversations`

서버에 새로운 엔드포인트 구현이 필요합니다.

**Request:**
```http
POST /api/conversations
Content-Type: application/json
X-API-Key: {api_key}

{
  "project_hash": "sha256_hash",
  "session_id": "abc123-def456",
  "entries": [
    {
      "line_index": 0,
      "type": "user",
      "message": { "role": "user", "content": "..." }
    },
    {
      "line_index": 1,
      "type": "assistant",
      "message": { "role": "assistant", "content": "..." }
    }
  ]
}
```

**Response (성공):**
```json
{
  "success": true,
  "entries_stored": 2
}
```

**중복 처리:**
- `session_id` + `line_index` 조합을 고유 키로 사용
- 동일한 키가 이미 존재하면 UPSERT (덮어쓰기)

---

## 6. 데이터 흐름

```
사용자가 프롬프트 입력
  │
  ▼
[UserPromptSubmit 훅]
  ├─ 파일 스캔 및 변경사항 감지
  ├─ pre-prompt 스냅샷 서버 전송
  └─ last_snapshot.json 저장 (files + transcript 보존)
  │
  ▼
Claude Code가 응답 생성 및 파일 수정
  │
  ▼
[Stop 훅]
  ├─ 파일 스캔 및 변경사항 감지
  ├─ post-prompt 스냅샷 서버 전송
  ├─ transcript_path에서 대화 히스토리 읽기
  │   └─ 이전 전송 이후의 새 엔트리만 읽기 (증분)
  ├─ 대화 엔트리 서버 전송 (POST /api/conversations)
  └─ last_snapshot.json 저장 (files + 새 transcript 상태)
```

---

## 7. Rewind 처리

Claude Code에서 대화를 되감기(Rewind)하면 transcript 파일의 내용이 줄어들 수 있습니다.

**감지 방법:**
```javascript
// 저장된 라인 번호가 실제 파일 라인 수보다 크거나 같으면 Rewind 발생
const effectiveStartLine = (startLine >= nonEmptyLines.length) ? 0 : startLine;
```

**처리:**
- Rewind 감지 시 처음부터 다시 읽어서 전송
- 서버에서 `session_id + line_index`로 UPSERT하여 중복 방지

---

## 8. 에러 처리

| 상황 | 처리 방식 |
|------|----------|
| `transcript_path` 없음 | 조용히 스킵 (대화 전송 건너뜀) |
| transcript 파일 없음 | 조용히 스킵 |
| JSONL 파싱 실패 | 해당 라인만 스킵, 다른 라인 계속 처리 |
| 서버 통신 실패 | 조용히 실패, `transcript` 상태 업데이트 안함 |
| 설정 파일 없음 | 기능 비활성화 (기존 동작 유지) |

**원칙:** 모든 에러는 조용히 처리되며, Claude Code의 정상 동작을 절대 방해하지 않습니다.

---

## 9. 제한사항

### 추적 불가능한 항목
- **모델 정보**: 훅에서 현재 사용 중인 모델을 정확히 알 수 없음
- **슬래시 명령어**: `/model`, `/clear` 등 내장 명령어는 훅을 거치지 않음

### 알려진 제한
- transcript 파일은 30일 비활성 후 Claude Code에 의해 자동 삭제됨
- 한 번에 최대 100개 엔트리만 전송 (설정 변경 가능)

---

## 10. 설정 예시

### 대화 추적 비활성화
```json
{
  "conversation_tracking": {
    "enabled": false
  }
}
```

### 전송 엔트리 수 조정
```json
{
  "conversation_tracking": {
    "enabled": true,
    "max_entries_per_request": 50
  }
}
```

---

## 11. 테스트 방법

### 수동 테스트

```bash
# Stop 훅 테스트 (transcript 전송)
echo '{"transcript_path":"/tmp/test.jsonl","timestamp":"2024-01-01T00:00:00Z"}' | \
  node .claude/hooks/stop.js

# 캐시 파일 확인
cat .codetracker/cache/last_snapshot.json
```

### 실제 환경 테스트

1. Claude Code 세션 시작
2. 프롬프트 입력 및 응답 대기
3. `.codetracker/cache/last_snapshot.json` 확인
   - `transcript` 필드에 `session_id`, `last_sent_line` 존재 확인
4. 서버 로그에서 `/api/conversations` 요청 확인

---

## 12. 버전 정보

- **구현 날짜**: 2024-01-15
- **config.json 버전**: 3.0
- **호환성**: 이전 버전 캐시 파일과 호환
