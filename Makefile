CXX := ccache g++

LLVM_CONFIG := llvm-config-19
CXX_FLAGS := $(shell $(LLVM_CONFIG) --cxxflags) -fno-rtti -O2
LD_FLAGS := $(shell $(LLVM_CONFIG) --ldflags)
LLVM_LIBS := $(shell $(LLVM_CONFIG) --libs)
# CLANG_LIBS := -lclang-cpp
CLANG_LIBS := -lclangTooling -lclangFrontend -lclangSerialization \
			-lclangDriver 	-lclangParse -lclangSema -lclangAnalysis \
			-lclangEdit -lclangAST -lclangLex -lclangBasic -lclangASTMatchers \
			-lclangAPINotes -lclangSupport

SQLITE_LIBS := -lsqlite3

GO := go
GOFLAGS := -ldflags "-s -w"
HOSTOS := linux
HOSTARCH := amd64

.PHONY: all clean

all: analyzer generator

clean:
	rm -rf bin/*

prepare:
	mkdir -p bin

analyzer: prepare
	$(CXX) $(CXX_FLAGS) src/analyzer/analyze.cpp \
		-o bin/analyzer \
		$(LD_FLAGS) \
		-Wl,--start-group $(CLANG_LIBS) -Wl,--end-group \
		$(LLVM_LIBS) \
		$(SQLITE_LIBS)

generator: prepare
	GOOS=$(HOSTOS) GOARCH=$(HOSTARCH) $(GO) build $(GOFLAGS) -o bin/generator src/generator/main.go