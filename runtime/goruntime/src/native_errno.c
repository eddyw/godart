#include <errno.h>
#include <poll.h>
#include <signal.h>
#include <unistd.h>
#include <stdint.h>
#include <sys/socket.h>

struct goruntime_result {
  intptr_t value;
  int error;
};

struct goruntime_result goruntime_connect(int fd, const void *address, uint32_t length) {
  struct goruntime_result result;
  result.value = connect(fd, (const struct sockaddr *)address, length);
  result.error = result.value < 0 ? errno : 0;
  return result;
}

struct goruntime_result goruntime_send(int fd, const void *data, size_t length, int flags) {
  struct goruntime_result result;
  result.value = send(fd, data, length, flags);
  result.error = result.value < 0 ? errno : 0;
  return result;
}

struct goruntime_result goruntime_sendto(int fd, const void *data, size_t length, int flags, const void *address, uint32_t address_length) {
  struct goruntime_result result;
  result.value = sendto(fd, data, length, flags, (const struct sockaddr *)address, address_length);
  result.error = result.value < 0 ? errno : 0;
  return result;
}

struct goruntime_result goruntime_recvfrom(int fd, void *data, size_t length, int flags, void *address, uint32_t *address_length) {
  struct goruntime_result result;
  result.value = recvfrom(fd, data, length, flags, (struct sockaddr *)address, (socklen_t *)address_length);
  result.error = result.value < 0 ? errno : 0;
  return result;
}

struct goruntime_result goruntime_getsockname(int fd, void *address, uint32_t *length) {
  struct goruntime_result result;
  result.value = getsockname(fd, (struct sockaddr *)address, (socklen_t *)length);
  result.error = result.value < 0 ? errno : 0;
  return result;
}

struct goruntime_result goruntime_getpeername(int fd, void *address, uint32_t *length) {
  struct goruntime_result result;
  result.value = getpeername(fd, (struct sockaddr *)address, (socklen_t *)length);
  result.error = result.value < 0 ? errno : 0;
  return result;
}

struct goruntime_result goruntime_unlinkat(int fd, const char *path, int flags) {
  struct goruntime_result result;
  result.value = unlinkat(fd, path, flags);
  result.error = result.value < 0 ? errno : 0;
  return result;
}

struct goruntime_result goruntime_kill(int pid, int signal) {
  struct goruntime_result result;
  result.value = kill(pid, signal);
  result.error = result.value < 0 ? errno : 0;
  return result;
}

struct goruntime_result goruntime_read(int fd, void *data, size_t length) {
  struct goruntime_result result;
  result.value = read(fd, data, length);
  result.error = result.value < 0 ? errno : 0;
  return result;
}

struct goruntime_result goruntime_write(int fd, const void *data, size_t length) {
  struct goruntime_result result;
  result.value = write(fd, data, length);
  result.error = result.value < 0 ? errno : 0;
  return result;
}

struct goruntime_result goruntime_poll(int fd, int write_ready) {
  struct pollfd entry = {fd, write_ready ? POLLOUT : POLLIN, 0};
  struct goruntime_result result;
  result.value = poll(&entry, 1, 0);
  result.error = result.value < 0 ? errno : 0;
  if (result.value > 0) {
    result.value = entry.revents;
    if (entry.revents & POLLNVAL) {
      result.value = -1;
      result.error = EBADF;
    }
  }
  return result;
}
