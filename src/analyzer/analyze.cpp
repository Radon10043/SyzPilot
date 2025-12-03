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
#include <clang/Lex/PPCallbacks.h>
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

struct AnalysisContext {
    std::vector<FuncInfo> funcs;
    std::vector<RecordInfo> records;
    std::vector<EnumInfo> enums;
    std::vector<TypedefInfo> typedefs;
    std::vector<GlobalVarInfo> globalVars;
    std::vector<MacroDefInfo> macros;
};

class MacroCallback : public PPCallbacks {
public:
    explicit MacroCallback(std::shared_ptr<AnalysisContext> ctx, SourceManager &sm) : ctx(ctx), sm(sm) {}

    void MacroDefined(const Token &macroNameTok, const MacroDirective *md) override {
        SourceLocation loc = macroNameTok.getLocation();

        /* skip invalid locations */
        if (loc.isInvalid())
            return;

        /* skip built-in macros */
        if (md->getMacroInfo()->isBuiltinMacro())
            return;

        /* skip macros from command line (e.g., -DXXXX) */
        if (sm.isWrittenInCommandLineFile(md->getLocation()))
            return;

        /* skip macros in built-in files */
        if (sm.isWrittenInBuiltinFile(md->getLocation()))
            return;

        /* skip includes */
        if (!sm.isInMainFile(loc))
            return;

        const MacroInfo *mi = md->getMacroInfo();
        if (!mi)
            return;

        /* TODO: get info of the macro */
        std::string name = macroNameTok.getIdentifierInfo()->getName().str();
        SourceRange sr = mi->getDefinitionLoc();
        std::string code =
            Lexer::getSourceText(CharSourceRange::getTokenRange(mi->getDefinitionLoc(), mi->getDefinitionEndLoc()), sm,
                                 LangOptions())
                .str();
        FullSourceLoc fsl = FullSourceLoc(loc, sm);
        std::string fp;
        int line = -1;
        if (fsl.isValid()) {
            SourceLocation sl = sm.getExpansionLoc(loc);
            FileID fid = sm.getFileID(sl);
            const FileEntry *fe = sm.getFileEntryForID(fid);
            fp = fe->tryGetRealPathName().str();
            line = fsl.getSpellingLineNumber();
        }

        /* add to context if macro name and code are not empty */
        if (!name.empty() && !code.empty()) {
            std::lock_guard<std::mutex> lock(DBMutex);
            code = "#define " + code;   /* add #define prefix to ensure the completeness */
            ctx->macros.push_back({name, fp, line, code});
        }
    }

private:
    std::shared_ptr<AnalysisContext> ctx;
    SourceManager &sm;
};

class MyASTVisitor : public RecursiveASTVisitor<MyASTVisitor> {
public:
    explicit MyASTVisitor(ASTContext *astCtx, std::shared_ptr<AnalysisContext> anaCtx)
        : astCtx(astCtx), anaCtx(anaCtx) {}

    bool VisitFunctionDecl(FunctionDecl *fd) {
        SourceManager &sm = astCtx->getSourceManager();

        /* skip includes */
        if (!sm.isInMainFile(fd->getBeginLoc()))
            return true;

        /* only process functions with a body */
        if (!fd->hasBody())
            return true;

        /* get info of the function */
        std::string fn = fd->getNameAsString();
        SourceRange sr = fd->getSourceRange();
        std::string code = Lexer::getSourceText(CharSourceRange::getTokenRange(sr), sm, astCtx->getLangOpts()).str();
        FullSourceLoc fsl = astCtx->getFullLoc(fd->getBeginLoc());
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
            anaCtx->funcs.push_back({fn, fp, line, code});

        return true;
    }

    bool VisitRecordDecl(RecordDecl *rd) {
        SourceManager &sm = astCtx->getSourceManager();

        /* skip includes */
        if (!sm.isInMainFile(rd->getBeginLoc()))
            return true;

        /* only process definitions */
        if (!rd->isThisDeclarationADefinition())
            return true;

        /* skip the anonymous records */
        if (rd->getNameAsString().empty())
            return true;

        /* get info of the record (struct or union) */
        std::string name = rd->getNameAsString();
        std::string type = rd->isStruct() ? "struct" : (rd->isUnion() ? "union" : "unknown");
        SourceRange sr = rd->getSourceRange();
        std::string code = Lexer::getSourceText(CharSourceRange::getTokenRange(sr), sm, astCtx->getLangOpts()).str();
        FullSourceLoc fsl = astCtx->getFullLoc(rd->getBeginLoc());
        std::string fp;
        int line = -1;
        if (fsl.isValid()) {
            SourceLocation sl = sm.getExpansionLoc(rd->getBeginLoc());
            FileID fid = sm.getFileID(sl);
            const FileEntry *fe = sm.getFileEntryForID(fid);
            fp = fe->tryGetRealPathName().str();
            line = fsl.getSpellingLineNumber();
        }

        /* add to vector if record name and code are not empty */
        if (!name.empty() && !code.empty())
            anaCtx->records.push_back({name, type, fp, line, code});

        return true;
    }

    bool VisitEnumDecl(EnumDecl *ed) {
        SourceManager &sm = astCtx->getSourceManager();

        /* skip includes */
        if (!sm.isInMainFile(ed->getBeginLoc()))
            return true;

        /* only process definitions */
        if (!ed->isThisDeclarationADefinition())
            return true;

        /* skip the anonymous enums */
        if (ed->getNameAsString().empty())
            return true;

        /* get info of the enum */
        std::string name = ed->getNameAsString();
        SourceRange sr = ed->getSourceRange();
        std::string code = Lexer::getSourceText(CharSourceRange::getTokenRange(sr), sm, astCtx->getLangOpts()).str();
        FullSourceLoc fsl = astCtx->getFullLoc(ed->getBeginLoc());
        std::string fp;
        int line = -1;
        if (fsl.isValid()) {
            SourceLocation sl = sm.getExpansionLoc(ed->getBeginLoc());
            FileID fid = sm.getFileID(sl);
            const FileEntry *fe = sm.getFileEntryForID(fid);
            fp = fe->tryGetRealPathName().str();
            line = fsl.getSpellingLineNumber();
        }

        /* add to vector if enum name and code are not empty */
        if (!name.empty() && !code.empty())
            anaCtx->enums.push_back({name, fp, line, code});

        return true;
    }

    bool VisitTypedefDecl(TypedefDecl *td) {
        SourceManager &sm = astCtx->getSourceManager();

        /* skip includes */
        if (!sm.isInMainFile(td->getBeginLoc()))
            return true;

        /* get info of the typedef */
        std::string typ = td->getUnderlyingType().getAsString();
        std::string def = td->getNameAsString();
        SourceRange sr = td->getSourceRange();
        std::string code = Lexer::getSourceText(CharSourceRange::getTokenRange(sr), sm, astCtx->getLangOpts()).str();
        FullSourceLoc fsl = astCtx->getFullLoc(td->getBeginLoc());
        std::string fp;
        int line = -1;
        if (fsl.isValid()) {
            SourceLocation sl = sm.getExpansionLoc(td->getBeginLoc());
            FileID fid = sm.getFileID(sl);
            const FileEntry *fe = sm.getFileEntryForID(fid);
            fp = fe->tryGetRealPathName().str();
            line = fsl.getSpellingLineNumber();
        }

        /* add to vector if old name (typ) and new name (def) are not empty */
        if (!typ.empty() && !def.empty())
            anaCtx->typedefs.push_back({typ, def, fp, line, code});

        return true;
    }

    bool VisitVarDecl(VarDecl *vd) {
        SourceManager &sm = astCtx->getSourceManager();

        /* skip includes */
        if (!sm.isInMainFile(vd->getBeginLoc()))
            return true;

        /* we only focus on global variables */
        if (!vd->hasGlobalStorage())
            return true;

        /* we only focus on global variables that are file variables */
        if (!vd->isFileVarDecl())
            return true;

        /* skip variables that are only declarations */
        if (vd->isThisDeclarationADefinition() == VarDecl::DeclarationOnly)
            return true;

        /* get info of the global variable */
        std::string name = vd->getNameAsString();
        std::string type = vd->getType().getAsString();
        FullSourceLoc fsl = astCtx->getFullLoc(vd->getBeginLoc());
        SourceRange sr = vd->getSourceRange();
        std::string code = Lexer::getSourceText(CharSourceRange::getTokenRange(sr), sm, astCtx->getLangOpts()).str();
        std::string fp;
        int line = -1;
        if (fsl.isValid()) {
            SourceLocation sl = sm.getExpansionLoc(vd->getBeginLoc());
            FileID fid = sm.getFileID(sl);
            const FileEntry *fe = sm.getFileEntryForID(fid);
            fp = fe->tryGetRealPathName().str();
            line = fsl.getSpellingLineNumber();
        }

        /* add to vector if name and code are not empty */
        if (!name.empty() && !code.empty())
            anaCtx->globalVars.push_back({name, fp, line, code});

        return true;
    }

private:
    ASTContext *astCtx;
    std::shared_ptr<AnalysisContext> anaCtx;
};

class MyASTConsumer : public ASTConsumer {
public:
    explicit MyASTConsumer(ASTContext *astCtx, std::shared_ptr<AnalysisContext> anaCtx)
        : visitor(astCtx, anaCtx), anaCtx(anaCtx) {}

    void HandleTranslationUnit(ASTContext &astCtx) override {
        visitor.TraverseDecl(astCtx.getTranslationUnitDecl());
        const auto &funcs = anaCtx->funcs;
        const auto &records = anaCtx->records;
        const auto &enums = anaCtx->enums;
        const auto &typedefs = anaCtx->typedefs;
        const auto &globalVars = anaCtx->globalVars;
        const auto &macros = anaCtx->macros;
        if (!DBMgr)
            return;

        /* insert functions, records, etc. into the database */
        std::lock_guard<std::mutex> lock(DBMutex);
        if (!funcs.empty())
            DBMgr->bulkInsertFuncs(funcs);
        if (!records.empty())
            DBMgr->bulkInsertRecords(records);
        if (!enums.empty())
            DBMgr->bulkInsertEnums(enums);
        if (!typedefs.empty())
            DBMgr->bulkInsertTypedefs(typedefs);
        if (!globalVars.empty())
            DBMgr->bulkInsertGlobalVars(globalVars);
        if (!macros.empty())
            DBMgr->bulkInsertMacroDefs(macros);
    }

private:
    MyASTVisitor visitor;
    std::shared_ptr<AnalysisContext> anaCtx;
};

class MyFrontendAction : public ASTFrontendAction {
public:
    MyFrontendAction() : anaCtx(std::make_shared<AnalysisContext>()) {}

    /* register callbacks for preprocessor */
    bool BeginSourceFileAction(CompilerInstance &ci) override {
        ci.getPreprocessor().addPPCallbacks(std::make_unique<MacroCallback>(anaCtx, ci.getSourceManager()));
        return true;
    }

    std::unique_ptr<ASTConsumer> CreateASTConsumer(CompilerInstance &ci, StringRef sr) override {
        return std::make_unique<MyASTConsumer>(&ci.getASTContext(), anaCtx);
    }

private:
    std::shared_ptr<AnalysisContext> anaCtx;
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