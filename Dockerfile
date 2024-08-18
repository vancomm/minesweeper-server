FROM golang:1.23.0@sha256:613a108a4a4b1dfb6923305db791a19d088f77632317cfc3446825c54fb862cd

WORKDIR /app

COPY . .

RUN go mod download

RUN go build -o server

EXPOSE 8000

CMD [ "./server", "-port", "8000" ]