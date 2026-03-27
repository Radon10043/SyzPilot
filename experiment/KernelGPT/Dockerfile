FROM ubuntu:22.04

SHELL [ "/bin/bash", "-c" ]

ENV GOPATH=/usr/local/go/bin
ENV LLVM_HOME=/llvm-15
ENV PATH=$GOPATH:$LLVM_HOME/bin:$PATH
ENV TZ=Asia/Shanghai

# Install basic packages
RUN apt-get update
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y tzdata
RUN apt-get install -y sudo wget curl git vim build-essential make gcc g++ flex bison bc debootstrap
RUN apt-get install -y qemu-system-x86 xz-utils cmake gdb libdw-dev libssl-dev sqlite3 libsqlite3-dev
RUN apt-get install -y rsync file devscripts equivs gnupg config-package-dev debhelper-compat dwarves
RUN apt-get install -y unzip gettext ccache git-email cpio gawk lsb-release software-properties-common
RUN apt-get install -y cloc openssh-server python3 python3-pip python3-dev language-pack-en-base jq
RUN apt-get install -y pkg-config libzstd-dev libxml2-dev zlib1g-dev sshfs bear

RUN pip install transformers rich loguru pyyaml openai dotenv psutil

# Install golang-1.24.4
RUN wget https://go.dev/dl/go1.24.4.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.24.4.linux-amd64.tar.gz && \
    rm go1.24.4.linux-amd64.tar.gz

# Install flatbuffers-23.5.26
RUN wget https://github.com/google/flatbuffers/archive/refs/tags/v23.5.26.tar.gz
RUN tar -xzvf v23.5.26.tar.gz && \
    pushd flatbuffers-23.5.26 && \
    cmake -G "Unix Makefiles" && \
    make -j && \
    make install && \
    popd && \
    rm -rf v23.5.26.tar.gz flatbuffers-23.5.26

# Install LLVM-15
RUN wget https://github.com/llvm/llvm-project/releases/download/llvmorg-15.0.0/clang+llvm-15.0.0-x86_64-linux-gnu-rhel-8.4.tar.xz
RUN mkdir /llvm-15 && tar -C /llvm-15 -xvf clang+llvm-15.0.0-x86_64-linux-gnu-rhel-8.4.tar.xz --strip-components=1
RUN rm clang+llvm-15.0.0-x86_64-linux-gnu-rhel-8.4.tar.xz
