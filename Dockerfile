FROM alpine:3.14
WORKDIR /app
COPY cmd/server/file-api-serverd /app/
RUN chmod +x file-api-serverd
EXPOSE 8085
CMD ["./file-api-serverd", "-addr=0.0.0.0:8085", "-pathfiles=/images/"]
