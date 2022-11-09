FROM golang:1.18-alpine
RUN apk add build-base
WORKDIR /app/
COPY . .
RUN mkdir reports
RUN apk --update --no-cache add postgresql-client
RUN apk --update --no-cache add make
EXPOSE 5000
RUN go mod download
ENTRYPOINT ["tail", "-f", "/dev/null"]
