CXX := ccache g++

LLVM_CONFIG := llvm-config-19
CXX_FLAGS := $(shell $(LLVM_CONFIG) --cxxflags) -fno-rtti -O2 -g
LD_FLAGS := $(shell $(LLVM_CONFIG) --ldflags)
LLVM_LIBS := $(shell $(LLVM_CONFIG) --libs)
# CLANG_LIBS := -lclang-cpp
CLANG_LIBS := -lclangTooling -lclangFrontend -lclangSerialization \
			-lclangDriver 	-lclangParse -lclangSema -lclangAnalysis \
			-lclangEdit -lclangAST -lclangLex -lclangBasic -lclangASTMatchers \
			-lclangAPINotes -lclangSupport

SQLITE_LIBS := -lsqlite3

.PHONY: all clean

all: analyzer

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