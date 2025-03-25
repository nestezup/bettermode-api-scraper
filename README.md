# BetterMode API Content Scraper

BetterMode API에서 게시물 콘텐츠를 가져오는 RESTful API 서버입니다. 게시물 ID를 입력하면 해당 게시물의 내용을 HTML 또는 텍스트 형식으로 반환합니다.

## 주요 기능

- BetterMode API에서 게시물 콘텐츠 추출
- HTML 또는 텍스트 형식으로 콘텐츠 반환
- 토큰 자동 갱신 기능
- Swagger 문서화
- CORS 지원

## 기술 스택

- Go 1.20+
- Chi 라우터
- Docker / Docker Compose

## 설치 및 실행 방법

### 로컬에서 실행

1. 의존성 설치:
```bash
go mod download
```

2. 애플리케이션 빌드:
```bash
go build -o bettermode-api main.go
```

3. 서버 실행:
```bash
./bettermode-api
```

기본적으로 포트 8080에서 실행됩니다. 환경 변수 `PORT`를 설정하여 다른 포트에서 실행할 수 있습니다.

### Docker로 실행

1. 애플리케이션 빌드:
```bash
go build -o bettermode-api main.go
```

2. Docker Compose로 실행:
```bash
docker-compose up -d
```

## API 사용 방법

### 게시물 콘텐츠 가져오기

**요청:**

```bash
curl -X POST http://localhost:8080/api/v1/content \
  -H "Content-Type: application/json" \
  -d '{"post_id": "rYDKVA8XqjSsqHK", "format": "text"}'
```

**응답:**

```json
{
  "content": "게시물 내용...",
  "format": "text",
  "post_id": "rYDKVA8XqjSsqHK",
  "title": "게시물 제목",
  "char_count": 12345
}
```

### 토큰 상태 확인

```bash
curl http://localhost:8080/api/v1/token/status
```

### 토큰 수동 갱신

```bash
curl http://localhost:8080/api/v1/token/refresh
```

## 배포 방법

### Docker Compose 사용

1. 애플리케이션 빌드:
```bash
go build -o bettermode-api main.go
```

2. Docker Compose로 배포:
```bash
docker-compose up -d
```

### 시스템 재부팅 후 자동 시작 설정

Docker와 서비스가 시스템 재부팅 후 자동으로 시작되도록 설정:

```bash
sudo systemctl enable docker
sudo systemctl enable bettermode-api  # Docker Compose 서비스를 systemd에 등록한 경우
```

## 라이센스

MIT 