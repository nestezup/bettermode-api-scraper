# 빌드 단계
FROM golang:1.20-alpine AS builder

# 필요한 시스템 도구 설치
RUN apk add --no-cache git

# 작업 디렉토리 설정
WORKDIR /app

# go.mod 및 go.sum 파일 복사 (종속성 다운로드용)
COPY go.mod go.sum ./

# 종속성 다운로드
RUN go mod download

# 소스 코드 복사
COPY . .

# 애플리케이션 빌드
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# 실행 단계
FROM alpine:latest

# 필요한 CA 인증서 설치 (HTTPS 요청용)
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# 빌드 단계에서 생성된 바이너리 복사
COPY --from=builder /app/main .
COPY --from=builder /app/docs ./docs

# 포트 지정
EXPOSE 8080

# 환경 변수 설정 (필요한 경우 추가)
ENV PORT=8080

# 실행 명령
CMD ["./main"] 