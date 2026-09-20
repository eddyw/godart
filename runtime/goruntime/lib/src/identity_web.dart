import 'slice.dart';

int $getuid() => -1;
int $geteuid() => -1;
int $getgid() => -1;
int $getegid() => -1;
int $getppid() => 0;
int $getpagesize() => 4096;

(Slice<int>, int) $getgroups() => (Slice<int>.nil(), -1);
