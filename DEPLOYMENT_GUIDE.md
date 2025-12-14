# CodeTracker Hooks 배포 가이드 (서버 관리자용)

사용자에게 플랫폼별 바이너리를 자동으로 배포하는 방법을 설명합니다.

## 배포 아키텍처

```
[웹서버]
    │
    ├── /static/hooks/
    │   ├── linux-amd64/
    │   │   ├── user_prompt_submit
    │   │   └── stop
    │   ├── linux-arm64/
    │   │   ├── user_prompt_submit
    │   │   └── stop
    │   ├── darwin-amd64/
    │   │   ├── user_prompt_submit
    │   │   └── stop
    │   ├── darwin-arm64/
    │   │   ├── user_prompt_submit
    │   │   └── stop
    │   └── windows-amd64/
    │       ├── user_prompt_submit.exe
    │       └── stop.exe
    │
    └── /api/download/setup
            │
            ▼
        플랫폼 감지 → 적절한 바이너리 포함 zip 생성

```

## 빌드 및 배포 파이프라인

### 1. 빌드 (CI/CD)

```bash
# GitHub Actions / GitLab CI 등에서
cd codetracker-hooks
make build-all
make release
```

### 2. 릴리스 아카이브 업로드

빌드 후 생성되는 파일:
```
dist/
├── codetracker-hooks-linux-amd64.tar.gz
├── codetracker-hooks-linux-arm64.tar.gz
├── codetracker-hooks-darwin-amd64.tar.gz
├── codetracker-hooks-darwin-arm64.tar.gz
└── codetracker-hooks-windows-amd64.zip
```

이 파일들을 서버의 `/static/hooks/` 디렉토리에 업로드합니다.

## 서버 측 다운로드 API 구현

### Python (Flask) 예시

```python
import os
import zipfile
import tempfile
from io import BytesIO
from flask import Flask, request, send_file, jsonify

app = Flask(__name__)

HOOKS_DIR = '/path/to/static/hooks'

def detect_platform(user_agent: str) -> tuple[str, str]:
    """User-Agent에서 플랫폼 감지"""
    ua = user_agent.lower()

    # OS 감지
    if 'windows' in ua:
        os_name = 'windows'
    elif 'mac' in ua or 'darwin' in ua:
        os_name = 'darwin'
    else:
        os_name = 'linux'

    # 아키텍처 감지
    if 'arm64' in ua or 'aarch64' in ua:
        arch = 'arm64'
    elif os_name == 'darwin' and ('m1' in ua or 'm2' in ua or 'm3' in ua):
        arch = 'arm64'
    else:
        arch = 'amd64'

    return os_name, arch

def get_binary_extension(os_name: str) -> str:
    """OS에 따른 바이너리 확장자"""
    return '.exe' if os_name == 'windows' else ''

def generate_settings_json(os_name: str) -> str:
    """플랫폼별 settings.json 생성"""
    if os_name == 'windows':
        return '''{
  "hooks": {
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": ".claude\\\\hooks\\\\user_prompt_submit.exe"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": ".claude\\\\hooks\\\\stop.exe"
          }
        ]
      }
    ]
  }
}'''
    else:
        return '''{
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
}'''

@app.route('/api/download/setup/<project_hash>')
def download_setup(project_hash: str):
    """프로젝트 설정 패키지 다운로드"""

    # 인증 확인 (생략)
    user = get_current_user()
    project = get_project_by_hash(project_hash)

    # 플랫폼 감지
    user_agent = request.headers.get('User-Agent', '')
    os_name, arch = detect_platform(user_agent)
    platform = f'{os_name}-{arch}'

    # 쿼리 파라미터로 플랫폼 오버라이드 가능
    if request.args.get('platform'):
        platform = request.args.get('platform')
        os_name = platform.split('-')[0]

    ext = get_binary_extension(os_name)
    hooks_path = os.path.join(HOOKS_DIR, platform)

    if not os.path.exists(hooks_path):
        return jsonify({'error': f'Unsupported platform: {platform}'}), 400

    # ZIP 파일 생성
    zip_buffer = BytesIO()
    with zipfile.ZipFile(zip_buffer, 'w', zipfile.ZIP_DEFLATED) as zf:

        # 1. .codetracker/config.json
        config = generate_config_json(project)
        zf.writestr('.codetracker/config.json', config)

        # 2. .codetracker/credentials.json
        credentials = generate_credentials_json(user, project)
        zf.writestr('.codetracker/credentials.json', credentials)

        # 3. .claude/settings.json
        settings = generate_settings_json(os_name)
        zf.writestr('.claude/settings.json', settings)

        # 4. 바이너리 파일들
        for binary in ['user_prompt_submit', 'stop']:
            binary_name = f'{binary}{ext}'
            binary_path = os.path.join(hooks_path, binary_name)

            if os.path.exists(binary_path):
                zf.write(binary_path, f'.claude/hooks/{binary_name}')

    zip_buffer.seek(0)

    return send_file(
        zip_buffer,
        mimetype='application/zip',
        as_attachment=True,
        download_name=f'codetracker-setup-{platform}.zip'
    )

def generate_config_json(project) -> str:
    """프로젝트 설정 JSON 생성"""
    import json
    return json.dumps({
        "version": "4.0",
        "server_url": "https://your-server.com",
        "ignore_patterns": [
            "*.pyc", "__pycache__", ".git", ".codetracker", ".claude",
            "node_modules", ".env", "*.log", ".DS_Store", "build/", "dist/"
        ],
        "track_extensions": [
            ".py", ".js", ".ts", ".jsx", ".tsx", ".java", ".cpp",
            ".c", ".h", ".go", ".rs", ".rb", ".php", ".md"
        ],
        "max_file_size": 1048576,
        "auto_snapshot": {
            "enabled": True,
            "min_interval_seconds": 30,
            "skip_patterns": ["^help", "^what is", "^explain"],
            "only_on_changes": True
        }
    }, indent=2)

def generate_credentials_json(user, project) -> str:
    """사용자 인증 정보 JSON 생성"""
    import json
    return json.dumps({
        "api_key": user.api_key,
        "username": user.username,
        "email": user.email,
        "current_project_hash": project.hash
    }, indent=2)
```

### Node.js (Express) 예시

```javascript
const express = require('express');
const archiver = require('archiver');
const path = require('path');
const fs = require('fs');

const app = express();
const HOOKS_DIR = '/path/to/static/hooks';

function detectPlatform(userAgent) {
  const ua = userAgent.toLowerCase();

  let os = 'linux';
  if (ua.includes('windows')) os = 'windows';
  else if (ua.includes('mac') || ua.includes('darwin')) os = 'darwin';

  let arch = 'amd64';
  if (ua.includes('arm64') || ua.includes('aarch64')) arch = 'arm64';
  else if (os === 'darwin' && (ua.includes('m1') || ua.includes('m2'))) arch = 'arm64';

  return { os, arch };
}

function generateSettingsJson(os) {
  const sep = os === 'windows' ? '\\\\' : '/';
  const ext = os === 'windows' ? '.exe' : '';

  return JSON.stringify({
    hooks: {
      UserPromptSubmit: [{
        hooks: [{
          type: 'command',
          command: `.claude${sep}hooks${sep}user_prompt_submit${ext}`
        }]
      }],
      Stop: [{
        hooks: [{
          type: 'command',
          command: `.claude${sep}hooks${sep}stop${ext}`
        }]
      }]
    }
  }, null, 2);
}

app.get('/api/download/setup/:projectHash', async (req, res) => {
  const { projectHash } = req.params;

  // 인증 및 프로젝트 조회 (생략)
  const user = await getCurrentUser(req);
  const project = await getProjectByHash(projectHash);

  // 플랫폼 감지
  const userAgent = req.headers['user-agent'] || '';
  let { os, arch } = detectPlatform(userAgent);

  if (req.query.platform) {
    [os, arch] = req.query.platform.split('-');
  }

  const platform = `${os}-${arch}`;
  const ext = os === 'windows' ? '.exe' : '';
  const hooksPath = path.join(HOOKS_DIR, platform);

  if (!fs.existsSync(hooksPath)) {
    return res.status(400).json({ error: `Unsupported platform: ${platform}` });
  }

  // ZIP 스트림 생성
  res.attachment(`codetracker-setup-${platform}.zip`);

  const archive = archiver('zip', { zlib: { level: 9 } });
  archive.pipe(res);

  // 설정 파일 추가
  archive.append(generateConfigJson(project), { name: '.codetracker/config.json' });
  archive.append(generateCredentialsJson(user, project), { name: '.codetracker/credentials.json' });
  archive.append(generateSettingsJson(os), { name: '.claude/settings.json' });

  // 바이너리 추가
  for (const binary of ['user_prompt_submit', 'stop']) {
    const binaryName = `${binary}${ext}`;
    const binaryPath = path.join(hooksPath, binaryName);

    if (fs.existsSync(binaryPath)) {
      archive.file(binaryPath, { name: `.claude/hooks/${binaryName}`, mode: 0o755 });
    }
  }

  archive.finalize();
});
```

## 프론트엔드 다운로드 UI

### 자동 플랫폼 감지

```html
<button id="download-btn" class="btn btn-primary">
  설정 파일 다운로드
</button>

<script>
document.getElementById('download-btn').addEventListener('click', () => {
  // 브라우저가 자동으로 User-Agent를 전송하므로
  // 서버에서 플랫폼을 감지함
  window.location.href = `/api/download/setup/${projectHash}`;
});
</script>
```

### 수동 플랫폼 선택

```html
<div class="platform-selector">
  <label>플랫폼 선택:</label>
  <select id="platform-select">
    <option value="">자동 감지</option>
    <option value="linux-amd64">Linux (x64)</option>
    <option value="linux-arm64">Linux (ARM64)</option>
    <option value="darwin-amd64">macOS (Intel)</option>
    <option value="darwin-arm64">macOS (Apple Silicon)</option>
    <option value="windows-amd64">Windows (x64)</option>
  </select>
  <button id="download-btn">다운로드</button>
</div>

<script>
document.getElementById('download-btn').addEventListener('click', () => {
  const platform = document.getElementById('platform-select').value;
  const url = platform
    ? `/api/download/setup/${projectHash}?platform=${platform}`
    : `/api/download/setup/${projectHash}`;
  window.location.href = url;
});
</script>
```

## GitHub Actions CI/CD 예시

```yaml
# .github/workflows/release.yml
name: Build and Release

on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Build all platforms
        run: make build-all

      - name: Create release archives
        run: make release

      - name: Upload to server
        run: |
          # SCP, rsync, S3 등으로 업로드
          rsync -avz dist/*.tar.gz dist/*.zip user@server:/path/to/static/hooks/

      - name: Create GitHub Release
        uses: softprops/action-gh-release@v1
        with:
          files: |
            dist/codetracker-hooks-*.tar.gz
            dist/codetracker-hooks-*.zip
```

## 버전 관리

### 바이너리 버전 확인

빌드 시 버전 정보가 포함됩니다:

```bash
./user_prompt_submit --version
# codetracker-hooks v1.0.0 (built 2024-01-01T00:00:00Z)
```

### 버전별 배포

```
/static/hooks/
├── latest/                    # 최신 버전 (심볼릭 링크)
├── v1.0.0/
│   ├── linux-amd64/
│   ├── darwin-arm64/
│   └── ...
└── v1.1.0/
    ├── linux-amd64/
    ├── darwin-arm64/
    └── ...
```

서버에서 버전 파라미터 지원:
```
/api/download/setup/{projectHash}?version=v1.0.0
```

## 보안 고려사항

1. **API 키 보호**: credentials.json은 HTTPS로만 전송
2. **바이너리 서명**: 프로덕션에서는 코드 서명 권장
3. **체크섬 제공**: 다운로드 페이지에 SHA256 체크섬 표시
4. **속도 제한**: 다운로드 API에 rate limiting 적용

## 체크섬 생성

```bash
cd dist
sha256sum *.tar.gz *.zip > checksums.txt
```

프론트엔드에서 표시:
```html
<div class="checksums">
  <h4>SHA256 Checksums</h4>
  <pre>
abc123... codetracker-hooks-linux-amd64.tar.gz
def456... codetracker-hooks-darwin-arm64.tar.gz
...
  </pre>
</div>
```
