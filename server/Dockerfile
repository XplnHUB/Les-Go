FROM alpine:latest
WORKDIR /root/
# Copy the binary that GoReleaser built and provided in the build context
COPY server_bin .
EXPOSE 8080
CMD ["./server_bin"]
