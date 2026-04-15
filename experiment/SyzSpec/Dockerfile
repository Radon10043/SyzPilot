# duplicate from https://github.com/seclab-ucr/SyzSpec/blob/1edbcffd6f56786d914b0c04458bee86abf215ac/Dockerfile
# update for generating syzlang specification and fuzzing for linux v6.18

FROM ghcr.io/klee/llvm:130_O_D_A_ubuntu_jammy-20230126 as llvm_base
FROM ghcr.io/klee/llvm:150_O_D_A_ubuntu_jammy-20230126 AS llvm_150
FROM ghcr.io/klee/gtest:1.11.0_ubuntu_jammy-20230126 as gtest_base
FROM ghcr.io/klee/uclibc:klee_uclibc_v1.3_130_ubuntu_jammy-20230126 as uclibc_base
FROM ghcr.io/klee/tcmalloc:2.9.1_ubuntu_jammy-20230126 as tcmalloc_base
FROM ghcr.io/klee/stp:2.3.3_ubuntu_jammy-20230126 as stp_base
FROM ghcr.io/klee/z3:4.8.15_ubuntu_jammy-20230126 as z3_base
FROM ghcr.io/klee/libcxx:130_ubuntu_jammy-20230126 as libcxx_base
FROM ghcr.io/klee/sqlite:3400100_ubuntu_jammy-20230126 as sqlite3_base
FROM llvm_base as intermediate
COPY --from=gtest_base /tmp /tmp/
COPY --from=uclibc_base /tmp /tmp/
COPY --from=tcmalloc_base /tmp /tmp/
COPY --from=stp_base /tmp /tmp/
COPY --from=z3_base /tmp /tmp/
COPY --from=libcxx_base /tmp /tmp/
COPY --from=sqlite3_base /tmp /tmp/
COPY --from=llvm_150 /tmp /tmp/
ENV COVERAGE=0
ENV USE_TCMALLOC=1
ENV BASE=/tmp
ENV LLVM_VERSION=14.0
ENV ENABLE_DOXYGEN=1
ENV ENABLE_OPTIMIZED=1
ENV ENABLE_DEBUG=1
ENV DISABLE_ASSERTIONS=0
ENV REQUIRES_RTTI=0
ENV SOLVERS=STP:Z3
ENV GTEST_VERSION=1.11.0
ENV UCLIBC_VERSION=klee_uclibc_v1.3
ENV TCMALLOC_VERSION=2.9.1
ENV SANITIZER_BUILD=
ENV STP_VERSION=2.3.3
ENV MINISAT_VERSION=master
ENV Z3_VERSION=4.8.15
ENV USE_LIBCXX=1
ENV KLEE_RUNTIME_BUILD="Debug+Asserts"
ENV SQLITE_VERSION=3400100
ENV GOPATH=/usr/local/go/bin
ENV TZ=Asia/Shanghai
LABEL maintainer="KLEE Developers"

# TODO remove adding sudo package
RUN apt update
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y tzdata
RUN apt-get install -y sudo emacs-nox vim-nox file python3-dateutil git
RUN apt-get install -y flex bison libtinfo5 libelf-dev bc build-essential
RUN apt-get install -y libssl-dev dwarves libdw-dev language-pack-en-base
RUN apt-get install -y qemu-system-x86 debootstrap jq
RUN rm -rf /var/lib/apt/lists/*

# Copy across source files needed for build
RUN git clone https://github.com/seclab-ucr/SyzSpec /tmp/klee_src && \
    git -C /tmp/klee_src checkout 1edbcffd6f56786d914b0c04458bee86abf215ac

WORKDIR /root
# Build klee
RUN /tmp/klee_src/scripts/build/build.sh --debug --install-system-deps klee \
    && pip3 install flask wllvm && \
    sudo rm -rf /var/lib/apt/lists/*


ENV PATH="$PATH:/tmp/llvm-130-install_O_D_A/bin:/root/klee_build/bin:/root/.local/bin"
ENV BASE=/tmp
# Add path to local LLVM installation - let system install precede local install
RUN /bin/bash -c 'echo "export \"PATH=$PATH:$(cd ${BASE}/llvm-*-install*/bin/ && pwd)\" >> /root/.bashrc"'

# Add KLEE header files to system standard include folder
RUN sudo /bin/bash -c 'ln -s /tmp/klee_src/include/klee /usr/include/'

ENV LD_LIBRARY_PATH /root/klee_build/lib/

# Add KLEE binary directory to PATH
RUN /bin/bash -c 'ln -s ${BASE}/klee_src /root/ && ln -s ${BASE}/klee_build* /root/klee_build'

# TODO Remove when STP is fixed
RUN /bin/bash -c 'echo "export LD_LIBRARY_PATH=$(cd ${BASE}/metaSMT-*-deps/stp-git-basic/lib/ && pwd):$LD_LIBRARY_PATH" >> /root/.bashrc'

# Install golang-1.24.4
RUN wget https://go.dev/dl/go1.24.4.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.24.4.linux-amd64.tar.gz && \
    rm go1.24.4.linux-amd64.tar.gz
RUN /bin/bash -c 'echo "export PATH=/usr/local/go/bin:$PATH" >> /root/.bashrc'

# Install LLVM-15
RUN wget https://github.com/llvm/llvm-project/releases/download/llvmorg-15.0.0/clang+llvm-15.0.0-x86_64-linux-gnu-rhel-8.4.tar.xz
RUN mkdir /llvm-15 && tar -C /llvm-15 -xvf clang+llvm-15.0.0-x86_64-linux-gnu-rhel-8.4.tar.xz --strip-components=1
RUN rm clang+llvm-15.0.0-x86_64-linux-gnu-rhel-8.4.tar.xz
