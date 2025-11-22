#include <iostream>
#include <memory>
#include <string>
#include <thread>
#include <vector>

#include <clang/AST/ASTConsumer.h>
#include <clang/AST/RecursiveASTVisitor.h>
#include <clang/Frontend/CompilerInstance.h>
#include <clang/Frontend/FrontendActions.h>
#include <clang/Lex/Lexer.h>
#include <clang/Tooling/CommonOptionsParser.h>
#include <clang/Tooling/JSONCompilationDatabase.h>
#include <clang/Tooling/Tooling.h>
#include <llvm/Support/CommandLine.h>
#include <llvm/Support/FileSystem.h>
#include <llvm/Support/Path.h>

#include "db.hpp"
#include <unistd.h>

using namespace clang;
using namespace clang::tooling;
using namespace llvm;

/* global variables */
static DatabaseManager *DBMgr = nullptr;
std::mutex DBMutex;

class MyASTVisitor : public RecursiveASTVisitor<MyASTVisitor> {
public:
    explicit MyASTVisitor(ASTContext *ctx) : ctx(ctx) {}

    std::vector<FuncInfo> &getFunctions() {
        return funcs;
    }

    bool VisitFunctionDecl(FunctionDecl *fd) {
        SourceManager &sm = ctx->getSourceManager();

        /* skip includes */
        if (!sm.isInMainFile(fd->getBeginLoc()))
            return true;

        /* only process functions with a body */
        if (!fd->hasBody())
            return true;

        /* get info of the function */
        std::string fn = fd->getNameAsString();
        SourceRange sr = fd->getSourceRange();
        std::string code = Lexer::getSourceText(CharSourceRange::getTokenRange(sr), sm, ctx->getLangOpts()).str();
        FullSourceLoc fsl = ctx->getFullLoc(fd->getBeginLoc());
        std::string fp;
        int line = -1;
        if (fsl.isValid()) {
            SourceLocation sl = sm.getExpansionLoc(fd->getBeginLoc());
            FileID fid = sm.getFileID(sl);
            const FileEntry *fe = sm.getFileEntryForID(fid);
            fp = fe->tryGetRealPathName().str();
            line = fsl.getSpellingLineNumber();
        }

        /* add to vector if function name and code are not empty */
        if (!fn.empty() && !code.empty())
            funcs.push_back({fn, fp, line, code});

        return true;
    }

private:
    ASTContext *ctx;
    std::vector<FuncInfo> funcs;
};

class MyASTConsumer : public ASTConsumer {
public:
    explicit MyASTConsumer(ASTContext *ctx) : visitor(ctx) {}

    void HandleTranslationUnit(ASTContext &ctx) override {
        visitor.TraverseDecl(ctx.getTranslationUnitDecl());
        const auto &funcs = visitor.getFunctions();
        if (funcs.empty())
            return;
        if (DBMgr) {
            std::lock_guard<std::mutex> lock(DBMutex);
            DBMgr->bulkInsertFuncs(funcs);
        }
    }

private:
    MyASTVisitor visitor;
};

class MyFrontendAction : public ASTFrontendAction {
public:
    std::unique_ptr<ASTConsumer> CreateASTConsumer(CompilerInstance &ci, StringRef sr) override {
        return std::make_unique<MyASTConsumer>(&ci.getASTContext());
    }
};

/**
 * worker thread function, for parallel analysis
 */
void workThread(const CompilationDatabase &compilations, std::vector<std::string> files) {
    ClangTool tool(compilations, files);
    tool.run(newFrontendActionFactory<MyFrontendAction>().get());
}

/**
 * command line options
 */
static cl::OptionCategory MyToolCategory("Kernel analyzer options");
static cl::opt<std::string> DatabasePath("i", cl::desc("path to compile_commands.json"), cl::value_desc("path"),
                                         cl::Required, cl::cat(MyToolCategory));
static cl::opt<int> ParallelJobs("j", cl::desc("number of parallel jobs (default: 1)"), cl::init(1),
                                 cl::cat(MyToolCategory));

int main(int argc, const char **argv) {
    /* parse command line options */
    cl::HideUnrelatedOptions(MyToolCategory);
    cl::ParseCommandLineOptions(argc, argv, "Kernel analyzer\n");

    /* TODO: specify database path via command line */
    DatabaseManager mgr("data/kernel.db");
    DBMgr = &mgr;

    /* load compile_commands.json specified by user */
    std::string err;
    auto compilations = JSONCompilationDatabase::loadFromFile(DatabasePath, err, JSONCommandLineSyntax::AutoDetect);
    if (!compilations) {
        errs() << "Error loading compilation database: " << err << "\n";
        return 1;
    }
    outs() << "Loaded compilation database from " << DatabasePath << "\n";

    /* we only focus on *.c and *.h */
    std::vector<std::string> allFiles = compilations->getAllFiles();
    std::vector<std::string> targetFiles;
    for (const auto &f : allFiles)
        if (f.find(".c") != std::string::npos || f.find(".h") != std::string::npos)
            targetFiles.push_back(f);

    /* switch work directory to compile_commands.json's directory to avoid errors like 'cannot open file xxx' */
    StringRef workdir = sys::path::parent_path(DatabasePath);
    outs() << "Changing working directory to " << workdir << "\n";
    if (chdir(workdir.str().c_str()) != 0) {
        errs() << "Failed to change directory to " << workdir << "\n";
    }

    /* split target files */
    size_t sz = targetFiles.size();
    int jobs = ParallelJobs;
    std::vector<std::thread> threads;
    size_t batchSize = sz / jobs;
    size_t rem = sz % jobs;
    size_t start = 0;
    for (int i = 0; i < jobs; i++) {
        size_t cnt = batchSize + (i < (int)rem ? 1 : 0);
        std::vector<std::string> threadFiles(targetFiles.begin() + start, targetFiles.begin() + start + cnt);
        threads.emplace_back(workThread, std::cref(*compilations), std::move(threadFiles));
        start += cnt;
    }

    /* start parallel analysis */
    for (auto &t : threads)
        if (t.joinable())
            t.join();

    /* wipe butt */
    DBMgr = nullptr;
    outs() << "Analysis completed.\n";
    return 0;
}