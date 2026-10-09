#define _GNU_SOURCE
#include <errno.h>
#include <fcntl.h>
#include <poll.h>
#include <pthread.h>
#include <signal.h>
#include <sys/stat.h>
#include <string.h>
#include <sys/ioctl.h>
#include <sys/wait.h>
#include <unistd.h>
#include <stdint.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <netinet/tcp.h>
#ifdef __linux__
#include <sys/syscall.h>
#elif defined(__APPLE__)
#include <sys/event.h>
#endif

struct goruntime_result {
  intptr_t value;
  int error;
};


struct goruntime_result goruntime_tcp_listen(const void *address, uint32_t length, int ipv6_only) {
  struct goruntime_result result = {-1, 0};
  const struct sockaddr *addr = address;
  int fd = socket(addr->sa_family, SOCK_STREAM, 0);
  if (fd < 0) { result.error = errno; return result; }
  int enabled = 1;
  if (fcntl(fd, F_SETFD, FD_CLOEXEC) < 0 ||
      fcntl(fd, F_SETFL, O_NONBLOCK) < 0 ||
      setsockopt(fd, SOL_SOCKET, SO_REUSEADDR, &enabled, sizeof(enabled)) < 0 ||
      (addr->sa_family == AF_INET6 && setsockopt(fd, IPPROTO_IPV6, IPV6_V6ONLY, &ipv6_only, sizeof(ipv6_only)) < 0) ||
      bind(fd, addr, length) < 0 || listen(fd, SOMAXCONN) < 0) {
    result.error = errno;
    close(fd);
    return result;
  }
  result.value = fd;
  return result;
}

struct goruntime_result goruntime_tcp_accept(int fd) {
  struct goruntime_result result = {-1, 0};
#ifdef __linux__
  int accepted = accept4(fd, NULL, NULL, SOCK_NONBLOCK | SOCK_CLOEXEC);
#else
  int accepted = accept(fd, NULL, NULL);
#endif
  if (accepted < 0) { result.error = errno; return result; }
  int enabled = 1;
  if (fcntl(accepted, F_SETFD, FD_CLOEXEC) < 0 ||
      fcntl(accepted, F_SETFL, O_NONBLOCK) < 0 ||
#ifdef __APPLE__
      setsockopt(accepted, SOL_SOCKET, SO_NOSIGPIPE, &enabled, sizeof(enabled)) < 0 ||
#endif
      setsockopt(accepted, IPPROTO_TCP, TCP_NODELAY, &enabled, sizeof(enabled)) < 0) {
    result.error = errno;
    close(accepted);
    return result;
  }
  result.value = accepted;
  return result;
}

struct goruntime_result goruntime_tcp_duplicate(int fd) {
  struct goruntime_result result = {-1, 0};
  result.value = fcntl(fd, F_DUPFD_CLOEXEC, 0);
  result.error = result.value < 0 ? errno : 0;
  return result;
}

struct goruntime_tcp_file {
  int value;
  int error;
  int operation;
};

struct goruntime_tcp_file goruntime_tcp_adopt(int fd) {
  struct goruntime_tcp_file result = {-1, 0, 1};
  struct goruntime_result copied = goruntime_tcp_duplicate(fd);
  if (copied.value < 0) { result.error = copied.error; return result; }
  int type = 0;
  socklen_t length = sizeof(type);
  result.operation = 2;
  if (getsockopt(copied.value, SOL_SOCKET, SO_TYPE, &type, &length) < 0) {
    result.error = errno;
    close(copied.value);
    return result;
  }
  result.operation = 0;
  if (type != SOCK_STREAM) {
    result.error = EINVAL;
    close(copied.value);
    return result;
  }
  result.value = copied.value;
  return result;
}

struct goruntime_result goruntime_setsockopt(int fd, int level, int option, const void *value, uint32_t length) {
  struct goruntime_result result;
  result.value = setsockopt(fd, level, option, value, length);
  result.error = result.value < 0 ? errno : 0;
  return result;
}

struct goruntime_result goruntime_shutdown(int fd, int direction) {
  struct goruntime_result result;
  result.value = shutdown(fd, direction);
  result.error = result.value < 0 ? errno : 0;
  return result;
}

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

struct goruntime_result goruntime_openat(int fd, const char *path, int flags, unsigned int mode) {
  struct goruntime_result result;
  result.value = openat(fd, path, flags, mode);
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

struct goruntime_process {
  int pid;
  int error;
  int input;
  int output;
  int diagnostic;
  int launch;
  int statusfd;
};

struct goruntime_process_state {
  int value;
  int error;
  int complete;
};

static int goruntime_process_pipe(int fd[2]) {
#ifdef __linux__
  if (pipe2(fd, O_CLOEXEC) < 0) return errno;
#else
  if (pipe(fd) < 0) return errno;
  for (int i = 0; i < 2; i++) {
    if (fcntl(fd[i], F_SETFD, FD_CLOEXEC) < 0) return errno;
  }
#endif
  // Sources above stdio avoid dup2 cycles when the caller closed stdio.
  for (int i = 0; i < 2; i++) {
    if (fd[i] < 3) {
      int moved = fcntl(fd[i], F_DUPFD_CLOEXEC, 3);
      if (moved < 0) return errno;
      close(fd[i]);
      fd[i] = moved;
    }
  }
  return 0;
}

static void goruntime_process_error(int fd, int code) {
  const char *bytes = (const char *)&code;
  size_t remaining = sizeof(code);
  while (remaining != 0) {
    ssize_t count = write(fd, bytes, remaining);
    if (count < 0 && errno == EINTR) continue;
    if (count <= 0) break;
    bytes += count;
    remaining -= count;
  }
  _exit(127);
}

struct goruntime_process goruntime_process_start(const char *path,
                                                char *const argv[],
                                                char *const env[],
                                                const char *directory,
                                                int pipe_ignored) {
  struct goruntime_process result = {-1, 0, -1, -1, -1, -1, -1};
  int pipes[5][2];
  for (int i = 0; i < 5; i++) pipes[i][0] = pipes[i][1] = -1;
  for (int i = 0; i < 5; i++) {
    result.error = goruntime_process_pipe(pipes[i]);
    if (result.error != 0) goto cleanup;
  }
  const int parent_ends[4] = {pipes[0][1], pipes[1][0], pipes[2][0], pipes[3][0]};
  for (int i = 0; i < 4; i++) {
    int flags = fcntl(parent_ends[i], F_GETFL);
    if (flags < 0 || fcntl(parent_ends[i], F_SETFL, flags | O_NONBLOCK) < 0) {
      result.error = errno;
      goto cleanup;
    }
  }
#ifdef __APPLE__
  result.statusfd = kqueue();
  if (result.statusfd < 0) {
    result.error = errno;
    goto cleanup;
  }
  if (result.statusfd < 3) {
    int moved = fcntl(result.statusfd, F_DUPFD_CLOEXEC, 3);
    if (moved < 0) {
      result.error = errno;
      goto cleanup;
    }
    close(result.statusfd);
    result.statusfd = moved;
  } else if (fcntl(result.statusfd, F_SETFD, FD_CLOEXEC) < 0) {
    result.error = errno;
    goto cleanup;
  }
#endif
  sigset_t blocked, saved;
  sigfillset(&blocked);
  result.error = pthread_sigmask(SIG_SETMASK, &blocked, &saved);
  if (result.error != 0) goto cleanup;
  pid_t pid = fork();
  int fork_error = errno;
  if (pid == 0) {
    // The child must not enter Dart or allocate before execve.
    close(pipes[4][1]);
    char ready;
    ssize_t count;
    do { count = read(pipes[4][0], &ready, 1); } while (count < 0 && errno == EINTR);
    if (count != 1) goruntime_process_error(pipes[3][1], EIO);
    close(pipes[4][0]);
    const int child_ends[3] = {pipes[0][0], pipes[1][1], pipes[2][1]};
    for (int i = 0; i < 3; i++) {
      if (dup2(child_ends[i], i) < 0) goruntime_process_error(pipes[3][1], errno);
    }
    for (int i = 0; i < 4; i++) {
      for (int j = 0; j < 2; j++) {
        if (pipes[i][j] != pipes[3][1]) close(pipes[i][j]);
      }
    }
    if (directory[0] != 0 && chdir(directory) < 0) {
      goruntime_process_error(pipes[3][1], errno);
    }
    struct sigaction action;
    memset(&action, 0, sizeof(action));
    sigemptyset(&action.sa_mask);
    action.sa_handler = SIG_DFL;
    for (int number = 1; number < NSIG; number++) {
      struct sigaction old;
      if (sigaction(number, NULL, &old) == 0 &&
          (number == SIGPIPE ||
           (old.sa_handler != SIG_IGN && old.sa_handler != SIG_DFL))) {
        action.sa_handler = number == SIGPIPE && pipe_ignored ? SIG_IGN : SIG_DFL;
        if (sigaction(number, &action, NULL) < 0) {
          goruntime_process_error(pipes[3][1], errno);
        }
      }
    }
    if (sigprocmask(SIG_SETMASK, &saved, NULL) < 0) {
      goruntime_process_error(pipes[3][1], errno);
    }
    execve(path, argv, env);
    goruntime_process_error(pipes[3][1], errno);
  }
  pthread_sigmask(SIG_SETMASK, &saved, NULL);
  if (pid < 0) {
    result.error = fork_error;
    goto cleanup;
  }
  result.pid = pid;
#ifdef __linux__
  // The barrier prevents an exit before pidfd_open establishes identity.
  result.statusfd = syscall(SYS_pidfd_open, pid, 0);
#elif defined(__APPLE__)
  // Register before the barrier release to retain status after another reaper.
  struct kevent event;
  EV_SET(&event, pid, EVFILT_PROC, EV_ADD, NOTE_EXIT | NOTE_EXITSTATUS, 0, NULL);
  if (kevent(result.statusfd, &event, 1, NULL, 0, NULL) < 0) {
    result.error = errno;
    goto cleanup;
  }
#endif
  close(pipes[4][0]);
  pipes[4][0] = -1;
  char ready = 1;
  ssize_t count;
  do { count = write(pipes[4][1], &ready, 1); } while (count < 0 && errno == EINTR);
  close(pipes[4][1]);
  pipes[4][1] = -1;
  result.input = pipes[0][1];
  result.output = pipes[1][0];
  result.diagnostic = pipes[2][0];
  result.launch = pipes[3][0];
  for (int i = 0; i < 4; i++) {
    close(pipes[i][i == 0 ? 0 : 1]);
  }
  return result;
cleanup:
  for (int i = 0; i < 5; i++) {
    for (int j = 0; j < 2; j++) if (pipes[i][j] >= 0) close(pipes[i][j]);
  }
  if (result.statusfd >= 0) close(result.statusfd);
  if (result.pid > 0) {
    int status;
    while (waitpid(result.pid, &status, 0) < 0 && errno == EINTR) {}
  }
  return result;
}

struct goruntime_process_state goruntime_process_wait(int pid, int statusfd) {
  struct goruntime_process_state result = {0, 0, 0};
  int status;
  pid_t waited = waitpid(pid, &status, WNOHANG);
  result.error = waited < 0 ? errno : 0;
  if (waited > 0) {
    result.value = status;
    result.complete = 1;
  }
#ifdef __linux__
  if (result.error == ECHILD && statusfd >= 0) {
    // PIDFD_GET_INFO version 0 has 64 bytes, with exit_code at byte 60.
    // Linux retains this actual status after another reaper consumes it.
    uint64_t info[8] = {1UL << 3};
    if (ioctl(statusfd, _IOWR(0xff, 11, info), info) == 0) {
      if (info[0] & (1UL << 3)) {
        memcpy(&status, (const char *)info + 60, sizeof(status));
        result.value = status;
        result.error = 0;
        result.complete = 1;
      } else {
        result.value = 0;
        result.error = 0;
      }
    }
  }
#elif defined(__APPLE__)
  if (result.error == ECHILD && statusfd >= 0) {
    struct kevent event;
    const struct timespec immediate = {0, 0};
    int count = kevent(statusfd, NULL, 0, &event, 1, &immediate);
    if (count == 1 && (event.fflags & NOTE_EXITSTATUS)) {
      result.value = (int)event.data;
      result.error = 0;
      result.complete = 1;
    } else if (count == 0) {
      result.error = 0;
    } else if (count < 0 && errno == EINTR) {
      result.error = EINTR;
    }
  }
#else
  (void)statusfd;
#endif
  return result;
}

struct goruntime_result goruntime_process_kill(int pid, int statusfd, int signal) {
  struct goruntime_result result;
#ifdef __linux__
  if (statusfd >= 0) {
    result.value = syscall(SYS_pidfd_send_signal, statusfd, signal, NULL, 0);
  } else {
    result.value = kill(pid, signal);
  }
#else
  (void)statusfd;
  result.value = kill(pid, signal);
#endif
  result.error = result.value < 0 ? errno : 0;
  return result;
}

struct goruntime_result goruntime_fstatat(int fd, const void *path, void *data, int flags) {
  struct goruntime_result result;
  result.value = fstatat(fd, (const char *)path, (struct stat *)data, flags);
  result.error = result.value < 0 ? errno : 0;
  return result;
}
