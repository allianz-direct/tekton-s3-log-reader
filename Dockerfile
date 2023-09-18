FROM public.ecr.aws/docker/library/golang:1.21-alpine as builder
WORKDIR /
ADD go.mod go.sum main.go /
ENV CGO_ENABLED=0
RUN go mod download
RUN go build
FROM public.ecr.aws/docker/library/alpine:latest
COPY --from=builder /tekton-s3-log-reader /
ENTRYPOINT ["/tekton-s3-log-reader"] 
