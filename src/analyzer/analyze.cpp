#include <iostream>
#include <string>

#include <clang/AST/ASTConsumer.h>
#include <clang/AST/RecursiveASTVisitor.h>
#include <clang/Frontend/CompilerInstance.h>
#include <clang/Frontend/FrontendActions.h>
#include <clang/Lex/Lexer.h>
#include <clang/Tooling/CommonOptionsParser.h>
#include <clang/Tooling/JSONCompilationDatabase.h>
#include <clang/Tooling/Tooling.h>
#include <llvm/Support/CommandLine.h>

#include "db.hpp"

using namespace clang;
using namespace clang::tooling;
using namespace llvm;

static DatabaseManager DBMgr("data/kernel.db");

class MyASTVisitor : public RecursiveASTVisitor<MyASTVisitor> {
public:
    explicit MyASTVisitor(ASTContext *ctx) : ctx(ctx) {}

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

        /* insert into database if function name and code are not empty */
        if (!fn.empty() && !code.empty())
            DBMgr.insertFunction(fn, fp, line, code);

        return true;
    }

private:
    ASTContext *ctx;
};

class MyASTConsumer : public ASTConsumer {
public:
    explicit MyASTConsumer(ASTContext *ctx) : visitor(ctx) {}

    void HandleTranslationUnit(ASTContext &ctx) override {
        visitor.TraverseDecl(ctx.getTranslationUnitDecl());
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
 * command line options
 */
static cl::OptionCategory MyToolCategory("Kernel analyzer options");
static cl::opt<std::string> DatabasePath("i", cl::desc("path to compile_commands.json"), cl::value_desc("path"),
                                         cl::Required, cl::cat(MyToolCategory));

int main(int argc, const char **argv) {
    /* parse command line options */
    cl::HideUnrelatedOptions(MyToolCategory);
    cl::ParseCommandLineOptions(argc, argv, "Kernel analyzer\n");

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

    /* start analyze */
    ClangTool tool(*compilations, targetFiles);
    return tool.run(newFrontendActionFactory<MyFrontendAction>().get());
}