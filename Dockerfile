# Use the official Golang image to create a build artifact.
# This is based on Debian and sets the GOPATH to /go.
# https://hub.docker.com/_/golang
FROM golang:1.24-alpine AS dev

# Install any necessary dependencies
RUN apk add -q --update \
    && apk add -q \
    bash \
    git \
    curl \
    make \
    gcc \
    g++ \
    musl-dev \
    nodejs \
    npm \
    sqlite-dev \
    && rm -rf /var/cache/apk/*

# Install air for hot-reloading
RUN curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Install pnpm
RUN npm install -g pnpm@9.4.0

# Install templ (Removed)
# RUN go install github.com/a-h/templ/cmd/templ@latest

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy everything from the current directory to the Working Directory inside the container
COPY . /app/

# Copy the git directory to ensure git commands work
# COPY .git /app/.git

# Removed incorrect Node.js dependency installation step for Go backend
# RUN rm -rf node_modules package-lock.json pnpm-lock.yaml && pnpm install --shamefully-hoist --strict-peer-dependencies=false --force

# Build the Go app
RUN CGO_ENABLED=0 go build -o /go/bin/app

# Add CMD for development
CMD ["sh", "-c", "air & wait"]

# Start a new stage from scratch
FROM gcr.io/distroless/static-debian11 AS prod

# Copy the binary to the production image from the builder stage.
COPY --from=dev /go/bin/app /app

# Copy the *.yaml file to the production image from the builder stage.
COPY --from=dev /app/etc/*.yaml /etc/

# Run the binary program produced by `go install`
CMD ["/app"]
