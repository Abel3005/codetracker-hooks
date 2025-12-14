# CodeTracker 설치 가이드 (Go 바이너리)

CodeTracker를 프로젝트에 설치하는 가이드입니다. Go 바이너리 버전은 **런타임 의존성이 없어** Node.js 설치가 필요하지 않습니다.

## 사전 요구사항

- **없음!** - Go 바이너리는 단독 실행 파일로 별도 런타임이 필요하지 않습니다.

## 설치 단계

### 1. 웹사이트에서 사용자 등록

1. CodeTracker 웹사이트에 접속
2. 계정 생성 및 로그인
3. 새 프로젝트 생성
4. **플랫폼 선택 후** 설정 파일 다운로드 (zip 파일)

다운로드한 파일에는 다음이 포함됩니다:
- `.codetracker/config.json` - 프로젝트 설정
- `.codetracker/credentials.json` - API 키 및 인  증 정보
- `.claude/hooks/user_prompt_submit` - 프롬프트 전 훅 (바이너리)
- `.claude/hooks/stop` - 프롬프트 후 훅 (바이너리)
- `.claude/settings.json` - Claude Code 훅 설정

### 2. 프로젝트에 파일 복사

다운로드한 zip 파일을 프로젝트 루트에 압축 해제:

```bash
cd your-project
unzip codetracker-setup.zip
```

압축 해제 후 디렉터리 구조:
```
your-project/
├── .codetracker/
│   ├── config.json          # 프로젝트 설정
│   ├── credentials.json     # API 키 (보안 유지!)
│   └── cache/               # 자동 생성됨
├── .claude/
│   ├── settings.json        # 훅 설정
│   └── hooks/
│       ├── user_prompt_submit   # Go 바이너리 (Unix/macOS)
│       ├── user_prompt_submit.exe  # Go 바이너리 (Windows)
│       ├── stop                 # Go 바이너리 (Unix/macOS)
│       └── stop.exe             # Go 바이너리 (Windows)
└── ... (your source files)
```

### 3. 실행 권한 설정 (Unix/macOS/Linux만)

```bash
chmod +x .claude/hooks/user_prompt_submit
chmod +x .claude/hooks/stop
```

Windows에서는 이 단계를 건너뛰세요.

### 4. .gitignore 업데이트

프로젝트의 `.gitignore` 파일에 다음을 추가:

```gitignore
# CodeTracker
.codetracker/credentials.json
.codetracker/cache/
```

**주의:** `credentials.json`은 절대 Git에 커밋하지 마세요!

### 5. 설치 테스트

#### 방법 1: 수동 테스트

**user_prompt_submit 테스트:**
```bash
echo '{"prompt":"test prompt","session_id":"test-123","timestamp":"2024-01-01T00:00:00Z"}' | \
  ./.claude/hooks/user_prompt_submit
```

성공하면 `.codetracker/cache/current_session.json` 파일이 생성됩니다:
```bash
cat .codetracker/cache/current_session.json
```

**stop 테스트:**
```bash
echo '{"timestamp":"2024-01-01T00:00:10Z"}' | \
  ./.claude/hooks/stop
```

성공하면 세션 파일이 삭제됩니다:
```bash
ls .codetracker/cache/  # current_session.json이 없어야 함
```

#### 방법 2: Claude Code로 실제 테스트

```bash
claude
```

Claude Code에서 간단한 프롬프트를 입력:
```
Create a new file called test.txt with "Hello World"
```

웹 대시보드에서 스냅샷과 상호작용이 기록되었는지 확인하세요.

## 설정 파일 구조

### `.codetracker/config.json`

```json
{
  "version": "4.0",
  "server_url": "https://your-codetracker-server.com",
  "ignore_patterns": [
    "*.pyc",
    "__pycache__",
    ".git",
    ".codetracker",
    ".claude",
    "node_modules",
    ".env",
    "*.log"
  ],
  "track_extensions": [
    ".py", ".js", ".ts", ".jsx", ".tsx",
    ".java", ".cpp", ".go", ".rs", ".md"
  ],
  "max_file_size": 1048576,
  "auto_snapshot": {
    "enabled": true,
    "min_interval_seconds": 30,
    "skip_patterns": ["^help", "^what is", "^explain"],
    "only_on_changes": true
  }
}
```

### `.codetracker/credentials.json`

```json
{
  "api_key": "your-api-key-here",
  "username": "your-username",
  "email": "your-email@example.com",
  "current_project_hash": "abc123..."
}
```

### `.claude/settings.json`

**Unix/macOS/Linux:**
```json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": ".claude/hooks/user_prompt_submit"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": ".claude/hooks/stop"
          }
        ]
      }
    ]
  }
}
```

**Windows:**
```json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": ".claude\\hooks\\user_prompt_submit.exe"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": ".claude\\hooks\\stop.exe"
          }
        ]
      }
    ]
  }
}
```

## 지원 플랫폼

| 운영체제 | 아키텍처 | 바이너리 파일명 |
|---------|---------|----------------|
| Linux | x64 (amd64) | `user_prompt_submit`, `stop` |
| Linux | ARM64 | `user_prompt_submit`, `stop` |
| macOS | x64 (Intel) | `user_prompt_submit`, `stop` |
| macOS | ARM64 (Apple Silicon) | `user_prompt_submit`, `stop` |
| Windows | x64 | `user_prompt_submit.exe`, `stop.exe` |

## 문제 해결

### 훅이 실행되지 않음

**문제:** Claude Code를 사용해도 스냅샷이 생성되지 않음

**해결 방법:**

1. **실행 권한 확인 (Unix/macOS/Linux):**
   ```bash
   ls -la .claude/hooks/
   ```
   `-rwxr-xr-x`와 같이 실행 권한(x)이 있어야 합니다.

   권한이 없으면:
   ```bash
   chmod +x .claude/hooks/user_prompt_submit
   chmod +x .claude/hooks/stop
   ```

2. **바이너리 실행 테스트:**
   ```bash
   ./.claude/hooks/user_prompt_submit --help
   ```
   실행되지 않으면 플랫폼이 맞는지 확인하세요.

3. **설정 파일 확인:**
   ```bash
   cat .claude/settings.json
   ```
   `hooks` 섹션이 올바르게 설정되어 있는지 확인

4. **플랫폼 확인:**
   ```bash
   uname -m  # Linux/macOS
   ```
   - `x86_64` → amd64 바이너리 사용
   - `aarch64` 또는 `arm64` → arm64 바이너리 사용

### 인증 오류

**문제:** 스냅샷이 서버에 저장되지 않음

**해결 방법:**

1. **credentials.json 확인:**
   ```bash
   cat .codetracker/credentials.json
   ```
   `api_key`와 `current_project_hash`가 있는지 확인

2. **서버 연결 테스트:**
   ```bash
   curl -H "X-API-Key: YOUR_API_KEY" \
     https://your-server.com/api/health
   ```

### Windows에서 경로 문제

**문제:** Windows에서 훅을 찾을 수 없음

**해결 방법:**

`.claude/settings.json`에서 백슬래시 사용:
```json
{
  "hooks": {
    "UserPromptSubmit": [{
      "hooks": [{
        "type": "command",
        "command": ".claude\\hooks\\user_prompt_submit.exe"
      }]
    }]
  }
}
```

또는 절대 경로 사용:
```json
{
  "command": "C:\\Users\\username\\project\\.claude\\hooks\\user_prompt_submit.exe"
}
```

### 스냅샷이 생성되지 않음

**문제:** 훅은 실행되지만 스냅샷이 기록되지 않음

**해결 방법:**

1. **파일 변경 확인:**
   `config.json`의 `auto_snapshot.only_on_changes`가 `true`이면 파일이 실제로 변경되어야 합니다.

2. **추적 확장자 확인:**
   변경한 파일의 확장자가 `track_extensions`에 포함되어 있는지 확인

3. **무시 패턴 확인:**
   파일이 `ignore_patterns`에 의해 무시되고 있지 않은지 확인

## Node.js 버전에서 마이그레이션

기존 Node.js 훅에서 Go 바이너리로 전환하는 경우:

1. **기존 훅 파일 삭제:**
   ```bash
   rm .claude/hooks/user_prompt_submit.js
   rm .claude/hooks/stop.js
   ```

2. **새 바이너리 다운로드 또는 복사:**
   웹사이트에서 새 설정 패키지를 다운로드하거나, 바이너리만 복사

3. **settings.json 업데이트:**
   ```json
   // 기존 (Node.js)
   "command": "node .claude/hooks/user_prompt_submit.js"

   // 변경 (Go)
   "command": ".claude/hooks/user_prompt_submit"
   ```

4. **실행 권한 설정:**
   ```bash
   chmod +x .claude/hooks/user_prompt_submit
   chmod +x .claude/hooks/stop
   ```

## 바이너리 직접 빌드 (개발자용)

소스 코드에서 직접 빌드하려면:

```bash
# 저장소 클론
git clone https://github.com/your-org/codetracker-hooks.git
cd codetracker-hooks

# 현재 플랫폼 빌드
make build

# 모든 플랫폼 빌드
make build-all

# 빌드 결과 확인
ls -la dist/
```

## 도움이 필요하신가요?

- **웹사이트:** https://your-codetracker-site.com
- **문서:** README.md, CLAUDE.md
- **이메일:** contact@thinktrace.net

Happy Coding!
