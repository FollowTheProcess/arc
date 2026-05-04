FROM gcr.io/distroless/static
ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/arc /usr/local/bin/arc
ENTRYPOINT [ "/usr/local/bin/arc" ]
