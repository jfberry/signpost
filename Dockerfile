FROM golang:1.25-alpine
WORKDIR /usr/src/app
COPY . .
RUN go build -o signpost .
EXPOSE 3035
CMD ["./signpost"]