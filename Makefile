CXX := ccache g++

LLVM_CONFIG := llvm-config-19
CXX_FLAGS :=
LD_FLAGS := $(shell $(LLVM_CONFIG) --ldflags)
LLVM_LIBS := $(shell $(LLVM_CONFIG) --libs)
# CLANG_LIBS := -lclang-cpp
CLANG_LIBS := -lclangTooling -lclangFrontend -lclangSerialization \
			-lclangDriver 	-lclangParse -lclangSema -lclangAnalysis \
			-lclangEdit -lclangAST -lclangLex -lclangBasic -lclangASTMatchers \
			-lclangAPINotes -lclangSupport

SQLITE_LIBS := -lsqlite3

GO := go
GOFLAGS :=
HOSTOS := linux
HOSTARCH := amd64

ifeq ("$(DEBUG)", "true")
	CXX_FLAGS := $(shell $(LLVM_CONFIG) --cxxflags) -fno-rtti -O0 -g
	GOFLAGS = -gcflags "all=-N -l"
else
	CXX_FLAGS := $(shell $(LLVM_CONFIG) --cxxflags) -fno-rtti -O2
	GOFLAGS := -ldflags "-s -w"
endif

.PHONY: all clean

all: analyzer generator syz-check syz-extract

clean:
	rm -rf bin/*

prepare:
	@mkdir -p bin

analyzer: prepare
	$(CXX) $(CXX_FLAGS) src/analyzer/analyze.cpp \
		-o bin/analyzer \
		$(LD_FLAGS) \
		-Wl,--start-group $(CLANG_LIBS) -Wl,--end-group \
		$(LLVM_LIBS) \
		$(SQLITE_LIBS)

generator: prepare
	GOOS=$(HOSTOS) GOARCH=$(HOSTARCH) $(GO) build $(GOFLAGS) -o bin/generator github.com/Radon10043/cloud/src/generator/

syz-check: prepare
	cd syzkaller/ && \
	GOOS=$(HOSTOS) GOARCH=$(HOSTARCH) $(GO) build $(GOFLAGS) -o $(PWD)/bin/syz-check $(PWD)/syzkaller/tools/syz-check/

syz-extract: prepare
	cd syzkaller/ && \
	make bin/syz-extract && \
	cp $(PWD)/syzkaller/bin/syz-extract $(PWD)/bin/syz-extract