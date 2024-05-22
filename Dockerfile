FROM golang:latest AS builder

WORKDIR /app

RUN export GO111MODULE=on

COPY ./go.mod ./go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -v -o ./tickets_manager ./cmd/tickets_manager

FROM alpine:latest AS runner

COPY --from=builder /app/tickets_manager .
COPY --from=builder /app/config.yaml ./config.yaml

EXPOSE 8880 5432 9092

CMD [ "./tickets_manager" ]
