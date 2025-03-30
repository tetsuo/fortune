FROM alpine:latest

# Define build arguments (defaults to linux/amd64)
ARG GOOS=linux
ARG GOARCH=amd64

# Install necessary dependencies
RUN apk add --no-cache \
    bash \
    ca-certificates \
    && addgroup -g 666 runner \
    && adduser -u 666 -G runner -h /home/runner -D runner

# Create and set ownership of the runner directory
RUN mkdir -p /home/runner/data && chown -R runner:runner /home/runner
VOLUME /home/runner
WORKDIR /home/runner

# Copy the correct binary dynamically based on GOOS and GOARCH
COPY --chown=runner:runner dist/frontend_${GOOS}_${GOARCH}_v1/bin/frontend-server /home/runner/

# Switch to non-root user
USER 666

# Run the application
CMD ["/home/runner/frontend-server"]
