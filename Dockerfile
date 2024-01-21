# build phase
FROM golang:1.20

WORKDIR /app
COPY . /app
ENV CGO_ENABLED=0

RUN go build -o tabu .

# execution phase
FROM alpine:latest

WORKDIR /
COPY --from=0 /app/tabu ./
COPY --from=0 /app/frontend ./frontend
COPY --from=0 /app/taboos.json ./taboos.json
EXPOSE 8080

CMD ["./tabu"]

