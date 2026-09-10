#!/bin/sh
# Invoke only in an authorized project as an ordinary member. No package repair,
# provider access, service/container creation or cleanup. Preserve the owned output on failure.
set -eu
: "${SODA_NATIVE_VALIDATE:?Set only on the authorized native validation target}"
test "$(id -u)" != 0
PATH=/usr/bin:/usr/sbin:/bin:/sbin
export PATH
for tool in gcc g++ make cmake ninja pkg-config readelf gdb strace patch diff file unzip zip xz bzip2 rsync ip ping dig lsof jq; do
    command -v "$tool" >/dev/null || { printf 'Missing native foundation tool: %s\n' "$tool" >&2; exit 1; }
done
umask 077
# Exercise an ordinary home checkout, not a potentially noexec/ephemeral /tmp.
work=$(mktemp -d "${HOME:?Project-local HOME required}/soda-foundation.XXXXXXXX")
printf 'Foundation evidence retained at %s\n' "$work"
mkdir "$work/home" "$work/source"
cat > "$work/source/main.c" <<'C'
#include <openssl/sha.h>
#include <zlib.h>
#include <stdio.h>
int main(void) {
    unsigned char digest[SHA256_DIGEST_LENGTH];
    if (!SHA256((const unsigned char *)"soda", 4, digest) || !zlibVersion()[0]) return 1;
    puts("foundation-c-ok");
    return 0;
}
C
cat > "$work/source/main.cpp" <<'CPP'
#include <numeric>
#include <vector>
#include <iostream>
int main() {
    const std::vector<int> values{1, 2, 3};
    if (std::accumulate(values.begin(), values.end(), 0) != 6) return 1;
    std::cout << "foundation-cxx-ok\n";
}
CPP
cat > "$work/source/CMakeLists.txt" <<'CMAKE'
cmake_minimum_required(VERSION 3.20)
project(soda_foundation LANGUAGES C CXX)
find_package(OpenSSL REQUIRED)
find_package(ZLIB REQUIRED)
add_executable(c_probe main.c)
target_link_libraries(c_probe PRIVATE OpenSSL::Crypto ZLIB::ZLIB)
add_executable(cxx_probe main.cpp)
set_property(TARGET cxx_probe PROPERTY CXX_STANDARD 17)
enable_testing()
add_test(NAME c_probe COMMAND c_probe)
add_test(NAME cxx_probe COMMAND cxx_probe)
CMAKE
# Do not read the member's CLI credentials, build presets, compiler overrides or
# debugger configuration. Same ordinary UID/groups, deliberately clean test HOME.
env -i HOME="$work/home" PATH=/usr/bin:/bin LANG=C.UTF-8 CC=/usr/bin/gcc CXX=/usr/bin/g++ \
    /usr/bin/cmake -S "$work/source" -B "$work/build" -G Ninja -DCMAKE_BUILD_TYPE=Debug > "$work/configure.log" 2>&1
env -i HOME="$work/home" PATH=/usr/bin:/bin LANG=C.UTF-8 /usr/bin/cmake --build "$work/build" > "$work/build.log" 2>&1
env -i HOME="$work/home" PATH=/usr/bin:/bin LANG=C.UTF-8 /usr/bin/ctest --test-dir "$work/build" --output-on-failure > "$work/test.log" 2>&1
env -i HOME="$work/home" PATH=/usr/bin:/bin LANG=C.UTF-8 /usr/bin/gdb --batch --nx --nh --return-child-result \
    -ex 'set auto-load off' -ex 'set debuginfod enabled off' -ex run --args "$work/build/c_probe" > "$work/debugger.log" 2>&1
grep -q '^foundation-c-ok' "$work/debugger.log"
env -i HOME="$work/home" PATH=/usr/bin:/bin LANG=C.UTF-8 /usr/bin/strace -o "$work/trace.log" /usr/bin/true
/usr/bin/readelf -h "$work/build/c_probe" > "$work/elf.txt"
printf 'Native C/C++ compile/link/check/debug/trace completed; not provider, workload or full-profile acceptance.\n'
