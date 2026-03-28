FROM golang:1.25-alpine
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o khayal ./cmd/khayal
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o kl ./cmd/kl
RUN mv khayal /usr/local/bin/khayal 
RUN mv kl /usr/local/bin/kl
EXPOSE 1133
CMD sh -c '[ -f /root/.config/khayal/config.yaml ] || /usr/local/bin/khayal init --vault /vault; /usr/local/bin/khayal start'
