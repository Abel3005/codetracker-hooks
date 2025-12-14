# CodeTracker Hooks (Go Binary)

Claude Code용 CodeTracker 훅의 Go 바이너리 구현입니다. Node.js 의존성 없이 단일 실행 파일로 동작합니다.

## 특징

- **런타임 의존성 없음** - Node.js 설치 불필요
- **빠른 실행** - ~5-10ms 시작 시간 (Node.js ~100-200ms 대비)
- **크로스 플랫폼** - Linux, macOS, Windows 지원
- **작은 바이너리** - 플랫폼당 ~5MB

## 지원 플랫폼

| OS | Architecture | Binary Size |
|----|--------------|-------------|
| Linux | amd64 | ~5.2 MB |
| Linux | arm64 | ~4.9 MB |
| macOS | amd64 (Intel) | ~5.3 MB |
| macOS | arm64 (Apple Silicon) | ~5.1 MB |
| Windows | amd64 | ~5.3 MB |

## 빌드

### 요구사항

- Go 1.21 이상

### 빌드 명령

```bash
# 현재 플랫폼
make build

# 모든 플랫폼
make build-all

# 릴리스 아카이브 생성
make release

# 테스트
make test

# 정리
make clean
```

### 빌드 출력

```
dist/
├── linux-amd64/
│   ├── user_prompt_submit
│   └── stop
├── linux-arm64/
│   ├── user_prompt_submit
│   └── stop
├── darwin-amd64/
│   ├── user_prompt_submit
│   └── stop
├── darwin-arm64/
│   ├── user_prompt_submit
│   └── stop
└── windows-amd64/
    ├── user_prompt_submit.exe
    └── stop.exe
```

## 프로젝트 구조

```
codetracker-hooks/
├── cmd/
│   ├── user_prompt_submit/     # 프롬프트 제출 전 훅
│   │   └── main.go
│   └── stop/                   # Claude 종료 후 훅
│       └── main.go
├── internal/
│   ├── config/                 # 설정 파일 로드
│   ├── gitignore/              # gitignore 패턴 매칭
│   ├── scanner/                # 파일 스캔 및 해시
│   ├── diff/                   # 변경 감지
│   ├── api/                    # HTTP 클라이언트
│   ├── session/                # 세션 파일 관리
│   └── cache/                  # 스냅샷 캐시 관리
├── go.mod
├── Makefile
├── INSTALLATION_GUIDE.md       # 사용자 설치 가이드
└── README.md
```

## 동작 방식

### user_prompt_submit

1. stdin에서 JSON 입력 받기 (`prompt`, `session_id`, `timestamp`)
2. `.codetracker/config.json` 및 `credentials.json` 로드
3. 프로젝트 파일 스캔 및 SHA256 해시 계산
4. 이전 스냅샷과 비교하여 변경 감지
5. 서버에 pre-prompt 스냅샷 생성 (`POST /api/snapshots`)
6. 세션 정보 저장 (`.codetracker/cache/current_session.json`)

### stop

1. stdin에서 JSON 입력 받기 (`timestamp`)
2. 세션 정보 로드
3. 프로젝트 파일 재스캔
4. 서버에 post-prompt 스냅샷 및 인터랙션 기록 (`POST /api/interactions`)
5. 세션 파일 삭제

## 설정

### `.claude/settings.json`

**Unix/macOS/Linux:**
```json
{
  "hooks": {
    "UserPromptSubmit": [{
      "hooks": [{
        "type": "command",
        "command": ".claude/hooks/user_prompt_submit"
      }]
    }],
    "Stop": [{
      "hooks": [{
        "type": "command",
        "command": ".claude/hooks/stop"
      }]
    }]
  }
}
```

**Windows:**
```json
{
  "hooks": {
    "UserPromptSubmit": [{
      "hooks": [{
        "type": "command",
        "command": ".claude\\hooks\\user_prompt_submit.exe"
      }]
    }],
    "Stop": [{
      "hooks": [{
        "type": "command",
        "command": ".claude\\hooks\\stop.exe"
      }]
    }]
  }
}
```

## 테스트

```bash
# user_prompt_submit 테스트
echo '{"prompt":"test","session_id":"123","timestamp":"2024-01-01T00:00:00Z"}' | \
  ./dist/user_prompt_submit

# stop 테스트
echo '{"timestamp":"2024-01-01T00:00:10Z"}' | \
  ./dist/stop
```

## 에러 처리

훅은 **silent fail** 패턴을 따릅니다:
- 모든 에러는 무시되고 exit code 0으로 종료
- Claude Code 실행을 절대 방해하지 않음
- panic도 recover로 처리

## 라이선스

MIT License
