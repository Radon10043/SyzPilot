FROM ubuntu:24.04

SHELL [ "/bin/bash", "-c" ]

ENV GOPATH=/usr/local/go/bin
ENV LLVM_HOME=/llvm-19
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
RUN apt-get install -y pkg-config libzstd-dev libxml2-dev zlib1g-dev sshfs tmux repo adb lsd
RUN apt-get install -y libncurses6 libtinfo6 ripgrep

RUN apt-get install -y bmake libarchive-tools libbsd-dev time libarchive-dev libarchive-tools libncurses5-dev

RUN apt-get install -y gcc-arm-linux-gnueabi gcc-aarch64-linux-gnu gcc-mips64el-linux-gnuabi64
RUN apt-get install -y gcc-powerpc64le-linux-gnu gcc-riscv64-linux-gnu gcc-s390x-linux-gnu

RUN echo "alias ls=lsd" >> /root/.bashrc

RUN pip install --break-system-packages compiledb

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

# Install LLVM-19
RUN wget https://github.com/llvm/llvm-project/releases/download/llvmorg-19.1.7/LLVM-19.1.7-Linux-X64.tar.xz
RUN mkdir /llvm-19 && tar -C /llvm-19 -xvf LLVM-19.1.7-Linux-X64.tar.xz --strip-components=1
RUN rm LLVM-19.1.7-Linux-X64.tar.xz

# Install cuttlefish
RUN curl -fsSL https://us-apt.pkg.dev/doc/repo-signing-key.gpg \
    -o /etc/apt/trusted.gpg.d/artifact-registry.asc
RUN chmod a+r /etc/apt/trusted.gpg.d/artifact-registry.asc
RUN echo "deb https://us-apt.pkg.dev/projects/android-cuttlefish-artifacts android-cuttlefish main" \
    | tee -a /etc/apt/sources.list.d/artifact-registry.list
RUN apt update
RUN apt-get install -y cuttlefish-base cuttlefish-user cuttlefish-orchestration
RUN groupadd --system kvm
RUN groupadd --system cvdnetwork
RUN groupadd --system render
RUN usermod -aG kvm,cvdnetwork,render root

# Use libncurses6 as libncurses5
RUN ln -s /usr/lib/x86_64-linux-gnu/libncurses.so.6 /usr/lib/x86_64-linux-gnu/libncurses.so.5
RUN ln -s /usr/lib/x86_64-linux-gnu/libtinfo.so.6 /usr/lib/x86_64-linux-gnu/libtinfo.so.5

# Install fzf
RUN git clone --depth 1 https://github.com/junegunn/fzf.git /root/.fzf && \
    yes | /root/.fzf/install
