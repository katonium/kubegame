# syntax=docker/dockerfile:1

# Deploy the application binary into a lean image
FROM gcr.io/distroless/base-debian11

COPY ./build/app /app

USER nonroot:nonroot

ENTRYPOINT ["/app"]