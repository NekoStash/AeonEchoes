FROM golang:1.26-alpine AS server-build
ARG GOPROXY=https://proxy.golang.org,direct
ARG GOSUMDB=sum.golang.org
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG NO_PROXY
ENV GOPROXY=${GOPROXY}
ENV GOSUMDB=${GOSUMDB}
ENV HTTP_PROXY=${HTTP_PROXY}
ENV HTTPS_PROXY=${HTTPS_PROXY}
ENV NO_PROXY=${NO_PROXY}
WORKDIR /src/apps/server
COPY apps/server/go.mod ./
RUN go mod download
COPY apps/server/ ./
RUN go build -o /out/aeon-server ./cmd/novelai

FROM alpine:3.21
WORKDIR /app
COPY --from=server-build /out/aeon-server /app/aeon-server
EXPOSE 8080
CMD ["/app/aeon-server"]
