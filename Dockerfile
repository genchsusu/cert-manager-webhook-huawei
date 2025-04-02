FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o webhook ./cmd/

FROM alpine:3

RUN apk add --no-cache tzdata \
    && ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

RUN apk add --no-cache bash curl ca-certificates && \
    echo 'export PS1="[docker \u@\h:\[\e[34;1m\]\w\[\033[m\] \[\033[1m\]\[\033[m\]] # "' >> /root/.bashrc && \
    echo 'alias ls="ls --color=auto"' >> /root/.bashrc && \
    echo 'alias ll="ls -l"' >> /root/.bashrc && \
    rm -rf /var/lib/apt/lists/* /var/cache/apk/*

COPY --from=builder /workspace/webhook /usr/local/bin/webhook

ENTRYPOINT ["webhook"]