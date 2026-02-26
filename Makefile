CXX := ccache clang++
CXX_FLAGS := $(shell pkg-config --cflags sqlite3) -pthread \
			-D__CLANG_RESOURCE_DIR__=\"$(shell clang -print-resource-dir)\"
CLANG_LIBS := -lclangTooling -lclangFrontend -lclangSerialization \
			-lclangDriver 	-lclangParse -lclangSema -lclangAnalysis \
			-lclangEdit -lclangAST -lclangLex -lclangBasic -lclangASTMatchers \
			-lclangAPINotes -lclangSupport

LLVM_CONFIG ?= llvm-config
LLVM_COMPONENTS := core support option demangle analysis bitreader profiledata target native
LLVM_COMPONENTS += frontendopenmp transformutils windowsdriver
LLVM_LIBS := $(shell $(LLVM_CONFIG) --libs $(LLVM_COMPONENTS) --system-libs)
LD_FLAGS := $(shell $(LLVM_CONFIG) --ldflags) $(shell pkg-config --libs sqlite3) -fuse-ld=lld

GO := go
GOFLAGS :=

TARGETOS ?= $(shell go env GOOS)

ifeq ($(shell echo $(TARGETOS) | tr '[:upper:]' '[:lower:]'),netbsd)
	CXX_FLAGS += -D__NETBSD_PATCH__
endif

ifeq ("$(DEBUG)", "true")
	CXX_FLAGS += $(shell $(LLVM_CONFIG) --cxxflags) -fno-rtti -O0 -g
	GOFLAGS += -gcflags "all=-N -l"
else
	CXX_FLAGS += $(shell $(LLVM_CONFIG) --cxxflags) -fno-rtti -O2
	GOFLAGS += -ldflags "-s -w"
endif

.PHONY: all clean

all: analyzer generator minitask syz-check syz-extract

clean:
	rm -rf bin

prepare:
	@mkdir -p bin

analyzer: prepare
	$(CXX) $(CXX_FLAGS) src/analyzer/analyze.cpp \
		-o bin/analyzer \
		$(LD_FLAGS) \
		-Wl,--start-group $(CLANG_LIBS) -Wl,--end-group \
		$(LLVM_LIBS)

generator: prepare
	$(GO) build $(GOFLAGS) -o bin/generator github.com/Radon10043/cloud/src/generator/

minitask: prepare
	$(GO) build $(GOFLAGS) -o bin/minitask github.com/Radon10043/cloud/src/minitask/

syz-check: prepare
	cd syzkaller/ && \
	$(GO) build $(GOFLAGS) -o $(PWD)/bin/syz-check $(PWD)/syzkaller/tools/syz-check/

syz-extract: prepare
	cd syzkaller/ && \
	$(GO) build $(GOFLAGS) -o $(PWD)/bin/syz-extract $(PWD)/syzkaller/sys/syz-extract/

pool2syz: prepare
	$(GO) build $(GOFLAGS) -o bin/pool2syz github.com/Radon10043/cloud/src/pool2syz/

refactor: prepare
	$(GO) build $(GOFLAGS) -o bin/refactor github.com/Radon10043/cloud/src/refactor/
