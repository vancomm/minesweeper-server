FROM golang:1.23.0@sha256:613a108a4a4b1dfb6923305db791a19d088f77632317cfc3446825c54fb862cd

WORKDIR /app

RUN go install github.com/jackc/tern/v2@latest

COPY . .

COPY ./bin/main.out server

EXPOSE 8000

CMD [ "./server", "-port", "8000" ]