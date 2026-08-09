#include <sqlite3.h>
#include <string>

#include <llvm/Support/raw_ostream.h>

struct FuncInfo {
    std::string name; /* function name */
    std::string file; /* file path     */
    int line;         /* line number   */
    std::string code; /* function code */
};

struct RecordInfo {
    std::string name; /* record name      */
    std::string type; /* struct or union  */
    std::string file; /* file path        */
    int line;         /* line number      */
    std::string code; /* record code      */
};

struct EnumInfo {
    std::string enumerator; /* enumerator name */
    std::string specifier;  /* specifier name  */
    std::string file;       /* file path       */
    int line;               /* line number     */
    std::string code;       /* enum code       */
};

struct TypedefInfo {
    std::string typ;  /* old name       */
    std::string def;  /* new name       */
    std::string file; /* file path      */
    int line;         /* line number    */
    std::string code; /* typedef code   */
};

struct GlobalVarInfo {
    std::string name; /* variable name  */
    std::string file; /* file path      */
    int line;         /* line number    */
    std::string code; /* variable code  */
};

struct MacroDefInfo {
    std::string name; /* macro name     */
    std::string file; /* file path      */
    int line;         /* line number    */
    std::string code; /* macro code     */
};

class DatabaseManager {
public:
    DatabaseManager() = default;

    DatabaseManager(const std::string &dbPath)
        : db(nullptr), insertFuncStmt(nullptr), insertRecStmt(nullptr), insertEnumStmt(nullptr),
          insertTypedefStmt(nullptr) {
        if (sqlite3_open(dbPath.c_str(), &db) != SQLITE_OK) {
            llvm::errs() << "Cannot open database: " << sqlite3_errmsg(db) << "\n";
            exit(1);
        }

        /* Enable WAL to speed up writes */
        sqlite3_exec(db, "PRAGMA synchronous = OFF; PRAGMA journal_mode = WAL;", nullptr, nullptr, nullptr);

        prepareFuncTable();
        prepareRecTable();
        prepareEnumTable();
        prepareTypedefTable();
        prepareGlobalVarTable();
        prepareMacroDefTable();
    }

    ~DatabaseManager() {
        sqlite3_finalize(insertFuncStmt);
        sqlite3_finalize(insertRecStmt);
        sqlite3_finalize(insertEnumStmt);
        sqlite3_finalize(insertTypedefStmt);
        sqlite3_finalize(insertGlobalVarStmt);
        sqlite3_finalize(insertMacroDefStmt);
        sqlite3_close(db);
    }

    /**
     * prepare functions table
     */
    void prepareFuncTable() {
        const char *tableSql = "CREATE TABLE IF NOT EXISTS functions ("
                               "id INTEGER PRIMARY KEY AUTOINCREMENT, "
                               "name TEXT NOT NULL, "
                               "file TEXT NOT NULL, "
                               "line INTEGER NOT NULL, "
                               "code TEXT NOT NULL, "
                               "UNIQUE(name, file, line));";
        if (sqlite3_exec(db, tableSql, 0, 0, 0) != SQLITE_OK) {
            llvm::errs() << "Failed to create 'functions' table: " << sqlite3_errmsg(db) << "\n";
            exit(1);
        }
        /* create an index so that we can boost query performance */
        const char *indexSql = "CREATE INDEX IF NOT EXISTS idxFuncName ON functions(name)";
        if (sqlite3_exec(db, indexSql, 0, 0, 0) != SQLITE_OK)
            llvm::errs() << "Failed to create index on 'functions' table: " << sqlite3_errmsg(db) << "\n";
        const char *insertSql = "INSERT OR IGNORE INTO functions (name, file, line, code) VALUES (?, ?, ?, ?);";
        if (sqlite3_prepare_v2(db, insertSql, -1, &insertFuncStmt, nullptr) != SQLITE_OK) {
            llvm::errs() << "Failed to prepare insert functions statement: " << sqlite3_errmsg(db) << "\n";
            exit(1);
        }
    }

    /**
     * prepare records table
     */
    void prepareRecTable() {
        const char *tableSql = "CREATE TABLE IF NOT EXISTS records ("
                               "id INTEGER PRIMARY KEY AUTOINCREMENT, "
                               "name TEXT NOT NULL, "
                               "type TEXT NOT NULL, "
                               "file TEXT NOT NULL, "
                               "line INTEGER NOT NULL, "
                               "code TEXT NOT NULL, "
                               "UNIQUE(name, file, line));";
        if (sqlite3_exec(db, tableSql, 0, 0, 0) != SQLITE_OK) {
            llvm::errs() << "Failed to create 'records' table: " << sqlite3_errmsg(db) << "\n";
            exit(1);
        }
        const char *indexSql = "CREATE INDEX IF NOT EXISTS idxRecName ON records(name)";
        if (sqlite3_exec(db, indexSql, 0, 0, 0) != SQLITE_OK)
            llvm::errs() << "Failed to create index on 'records' table: " << sqlite3_errmsg(db) << "\n";
        const char *insertSql = "INSERT OR IGNORE INTO records (name, type, file, line, code) VALUES (?, ?, ?, ?, ?);";
        if (sqlite3_prepare_v2(db, insertSql, -1, &insertRecStmt, nullptr) != SQLITE_OK) {
            llvm::errs() << "Failed to prepare insert records statement: " << sqlite3_errmsg(db) << "\n";
            exit(1);
        }
    }

    /**
     * prepare enums table
     */
    void prepareEnumTable() {
        const char *tableSql = "CREATE TABLE IF NOT EXISTS enums ("
                               "id INTEGER PRIMARY KEY AUTOINCREMENT, "
                               "enumerator TEXT NOT NULL, "
                               "specifier TEXT NOT NULL, "
                               "file TEXT NOT NULL, "
                               "line INTEGER NOT NULL, "
                               "code TEXT NOT NULL, "
                               "UNIQUE(enumerator, file, line));";
        if (sqlite3_exec(db, tableSql, 0, 0, 0) != SQLITE_OK) {
            llvm::errs() << "Failed to create 'enums' table: " << sqlite3_errmsg(db) << "\n";
            exit(1);
        }
        const char *indexSql = "CREATE INDEX IF NOT EXISTS idxEnumerator ON enums(enumerator)";
        if (sqlite3_exec(db, indexSql, 0, 0, 0) != SQLITE_OK)
            llvm::errs() << "Failed to create index on 'enums' table: " << sqlite3_errmsg(db) << "\n";
        const char *insertSql = "INSERT OR IGNORE INTO enums (enumerator, specifier, file, line, code) VALUES (?, ?, ?, ?, ?);";
        if (sqlite3_prepare_v2(db, insertSql, -1, &insertEnumStmt, nullptr) != SQLITE_OK) {
            llvm::errs() << "Failed to prepare insert enums statement: " << sqlite3_errmsg(db) << "\n";
            exit(1);
        }
    }

    /**
     * prepare typedefs table
     */
    void prepareTypedefTable() {
        const char *tableSql = "CREATE TABLE IF NOT EXISTS typedefs ("
                               "id INTEGER PRIMARY KEY AUTOINCREMENT, "
                               "type TEXT NOT NULL, "
                               "define TEXT NOT NULL, "
                               "file TEXT NOT NULL, "
                               "line INTEGER NOT NULL, "
                               "code TEXT NOT NULL, "
                               "UNIQUE(type, define, file, line));";
        if (sqlite3_exec(db, tableSql, 0, 0, 0) != SQLITE_OK) {
            llvm::errs() << "Failed to create 'typedefs' table: " << sqlite3_errmsg(db) << "\n";
            exit(1);
        }
        const char *indexSql = "CREATE INDEX IF NOT EXISTS idxTypedefDefine ON typedefs(define)";
        if (sqlite3_exec(db, indexSql, 0, 0, 0) != SQLITE_OK)
            llvm::errs() << "Failed to create index on 'typedefs' table: " << sqlite3_errmsg(db) << "\n";
        const char *insertSql =
            "INSERT OR IGNORE INTO typedefs (type, define, file, line, code) VALUES (?, ?, ?, ?, ?);";
        if (sqlite3_prepare_v2(db, insertSql, -1, &insertTypedefStmt, nullptr) != SQLITE_OK) {
            llvm::errs() << "Failed to prepare insert typedefs statement: " << sqlite3_errmsg(db) << "\n";
            exit(1);
        }
    }

    /**
     * prepare global variables table
     */
    void prepareGlobalVarTable() {
        const char *tableSql = "CREATE TABLE IF NOT EXISTS globalVars ("
                               "id INTEGER PRIMARY KEY AUTOINCREMENT, "
                               "name TEXT NOT NULL, "
                               "file TEXT NOT NULL, "
                               "line INTEGER NOT NULL, "
                               "code TEXT NOT NULL, "
                               "UNIQUE(name, file, line));";
        if (sqlite3_exec(db, tableSql, 0, 0, 0) != SQLITE_OK) {
            llvm::errs() << "Failed to create 'globalVars' table: " << sqlite3_errmsg(db) << "\n";
            exit(1);
        }
        const char *indexSql = "CREATE INDEX IF NOT EXISTS idxGlobalVarName ON globalVars(name)";
        if (sqlite3_exec(db, indexSql, 0, 0, 0) != SQLITE_OK)
            llvm::errs() << "Failed to create index on 'globalVars' table: " << sqlite3_errmsg(db) << "\n";
        const char *insertSql = "INSERT OR IGNORE INTO globalVars (name, file, line, code) VALUES (?, ?, ?, ?);";
        if (sqlite3_prepare_v2(db, insertSql, -1, &insertGlobalVarStmt, nullptr) != SQLITE_OK) {
            llvm::errs() << "Failed to prepare insert global variables statement: " << sqlite3_errmsg(db) << "\n";
            exit(1);
        }
    }

    /**
     * prepare macro definitions table
     */
    void prepareMacroDefTable() {
        const char *tableSql = "CREATE TABLE IF NOT EXISTS macroDefs ("
                               "id INTEGER PRIMARY KEY AUTOINCREMENT, "
                               "name TEXT NOT NULL, "
                               "file TEXT NOT NULL, "
                               "line INTEGER NOT NULL, "
                               "code TEXT NOT NULL,"
                               "UNIQUE(name, file, line));";
        if (sqlite3_exec(db, tableSql, 0, 0, 0) != SQLITE_OK) {
            llvm::errs() << "Failed to create 'macroDefs' table: " << sqlite3_errmsg(db) << "\n";
            exit(1);
        }
        const char *indexSql = "CREATE INDEX IF NOT EXISTS idxMacroDefName ON macroDefs(name)";
        if (sqlite3_exec(db, indexSql, 0, 0, 0) != SQLITE_OK)
            llvm::errs() << "Failed to create index on 'macroDefs' table: " << sqlite3_errmsg(db) << "\n";
        const char *insertSql = "INSERT OR IGNORE INTO macroDefs (name, file, line, code) VALUES (?, ?, ?, ?);";
        if (sqlite3_prepare_v2(db, insertSql, -1, &insertMacroDefStmt, nullptr) != SQLITE_OK) {
            llvm::errs() << "Failed to prepare insert macro definitions statement: " << sqlite3_errmsg(db) << "\n";
            exit(1);
        }
    }

    /**
     * insert functions into the database
     */
    void bulkInsertFuncs(const std::vector<FuncInfo> &funcs) {
        char *err = nullptr;
        sqlite3_exec(db, "BEGIN TRANSACTION;", nullptr, nullptr, nullptr);
        for (auto &func : funcs) {
            sqlite3_bind_text(insertFuncStmt, 1, func.name.c_str(), -1, SQLITE_STATIC);
            sqlite3_bind_text(insertFuncStmt, 2, func.file.c_str(), -1, SQLITE_STATIC);
            sqlite3_bind_int(insertFuncStmt, 3, func.line);
            sqlite3_bind_text(insertFuncStmt, 4, func.code.c_str(), -1, SQLITE_STATIC);
            if (sqlite3_step(insertFuncStmt) != SQLITE_DONE)
                llvm::errs() << "Insert functions error: " << sqlite3_errmsg(db) << "\n";
            sqlite3_reset(insertFuncStmt);
        }
        if (sqlite3_exec(db, "COMMIT;", nullptr, nullptr, &err) != SQLITE_OK) {
            llvm::errs() << "Commit functions error: " << err << "\n";
            sqlite3_free(err);
        }
    }

    /**
     * insert records into the database
     */
    void bulkInsertRecords(const std::vector<RecordInfo> &records) {
        char *err = nullptr;
        sqlite3_exec(db, "BEGIN TRANSACTION;", nullptr, nullptr, nullptr);
        for (auto &record : records) {
            sqlite3_bind_text(insertRecStmt, 1, record.name.c_str(), -1, SQLITE_STATIC);
            sqlite3_bind_text(insertRecStmt, 2, record.type.c_str(), -1, SQLITE_STATIC);
            sqlite3_bind_text(insertRecStmt, 3, record.file.c_str(), -1, SQLITE_STATIC);
            sqlite3_bind_int(insertRecStmt, 4, record.line);
            sqlite3_bind_text(insertRecStmt, 5, record.code.c_str(), -1, SQLITE_STATIC);
            if (sqlite3_step(insertRecStmt) != SQLITE_DONE)
                llvm::errs() << "Insert records error: " << sqlite3_errmsg(db) << "\n";
            sqlite3_reset(insertRecStmt);
        }
        if (sqlite3_exec(db, "COMMIT;", nullptr, nullptr, &err) != SQLITE_OK) {
            llvm::errs() << "Commit records error: " << err << "\n";
            sqlite3_free(err);
        }
    }

    /**
     * insert enums into the database
     */
    void bulkInsertEnums(const std::vector<EnumInfo> &enums) {
        char *err = nullptr;
        sqlite3_exec(db, "BEGIN TRANSACTION;", nullptr, nullptr, nullptr);
        for (auto &enm : enums) {
            sqlite3_bind_text(insertEnumStmt, 1, enm.enumerator.c_str(), -1, SQLITE_STATIC);
            sqlite3_bind_text(insertEnumStmt, 2, enm.specifier.c_str(), -1, SQLITE_STATIC);
            sqlite3_bind_text(insertEnumStmt, 3, enm.file.c_str(), -1, SQLITE_STATIC);
            sqlite3_bind_int(insertEnumStmt, 4, enm.line);
            sqlite3_bind_text(insertEnumStmt, 5, enm.code.c_str(), -1, SQLITE_STATIC);
            if (sqlite3_step(insertEnumStmt) != SQLITE_DONE)
                llvm::errs() << "Insert enums error: " << sqlite3_errmsg(db) << "\n";
            sqlite3_reset(insertEnumStmt);
        }
        if (sqlite3_exec(db, "COMMIT;", nullptr, nullptr, &err) != SQLITE_OK) {
            llvm::errs() << "Commit enums error: " << err << "\n";
            sqlite3_free(err);
        }
    }

    /**
     * insert typedefs into the database
     */
    void bulkInsertTypedefs(const std::vector<TypedefInfo> &typedefs) {
        char *err = nullptr;
        sqlite3_exec(db, "BEGIN TRANSACTION;", nullptr, nullptr, nullptr);
        for (auto &td : typedefs) {
            sqlite3_bind_text(insertTypedefStmt, 1, td.typ.c_str(), -1, SQLITE_STATIC);
            sqlite3_bind_text(insertTypedefStmt, 2, td.def.c_str(), -1, SQLITE_STATIC);
            sqlite3_bind_text(insertTypedefStmt, 3, td.file.c_str(), -1, SQLITE_STATIC);
            sqlite3_bind_int(insertTypedefStmt, 4, td.line);
            sqlite3_bind_text(insertTypedefStmt, 5, td.code.c_str(), -1, SQLITE_STATIC);
            if (sqlite3_step(insertTypedefStmt) != SQLITE_DONE)
                llvm::errs() << "Insert typedefs error: " << sqlite3_errmsg(db) << "\n";
            sqlite3_reset(insertTypedefStmt);
        }
        if (sqlite3_exec(db, "COMMIT;", nullptr, nullptr, &err) != SQLITE_OK) {
            llvm::errs() << "Commit typedefs error: " << err << "\n";
            sqlite3_free(err);
        }
    }

    /**
     * insert global variables into the database
     */
    void bulkInsertGlobalVars(const std::vector<GlobalVarInfo> &globalVars) {
        char *err = nullptr;
        sqlite3_exec(db, "BEGIN TRANSACTION;", nullptr, nullptr, nullptr);
        for (auto &gv : globalVars) {
            sqlite3_bind_text(insertGlobalVarStmt, 1, gv.name.c_str(), -1, SQLITE_STATIC);
            sqlite3_bind_text(insertGlobalVarStmt, 2, gv.file.c_str(), -1, SQLITE_STATIC);
            sqlite3_bind_int(insertGlobalVarStmt, 3, gv.line);
            sqlite3_bind_text(insertGlobalVarStmt, 4, gv.code.c_str(), -1, SQLITE_STATIC);
            if (sqlite3_step(insertGlobalVarStmt) != SQLITE_DONE)
                llvm::errs() << "Insert global variables error: " << sqlite3_errmsg(db) << "\n";
            sqlite3_reset(insertGlobalVarStmt);
        }
        if (sqlite3_exec(db, "COMMIT;", nullptr, nullptr, &err) != SQLITE_OK) {
            llvm::errs() << "Commit global variables error: " << err << "\n";
            sqlite3_free(err);
        }
    }

    /**
     * insert macro definitions into the database
     */
    void bulkInsertMacroDefs(const std::vector<MacroDefInfo> &macroDefs) {
        char *err = nullptr;
        sqlite3_exec(db, "BEGIN TRANSACTION;", nullptr, nullptr, nullptr);
        for (auto &md : macroDefs) {
            sqlite3_bind_text(insertMacroDefStmt, 1, md.name.c_str(), -1, SQLITE_STATIC);
            sqlite3_bind_text(insertMacroDefStmt, 2, md.file.c_str(), -1, SQLITE_STATIC);
            sqlite3_bind_int(insertMacroDefStmt, 3, md.line);
            sqlite3_bind_text(insertMacroDefStmt, 4, md.code.c_str(), -1, SQLITE_STATIC);
            if (sqlite3_step(insertMacroDefStmt) != SQLITE_DONE)
                llvm::errs() << "Insert macro definitions error: " << sqlite3_errmsg(db) << "\n";
            sqlite3_reset(insertMacroDefStmt);
        }
        if (sqlite3_exec(db, "COMMIT;", nullptr, nullptr, &err) != SQLITE_OK) {
            llvm::errs() << "Commit macro definitions error: " << err << "\n";
            sqlite3_free(err);
        }
    }

private:
    sqlite3 *db;
    sqlite3_stmt *insertFuncStmt;
    sqlite3_stmt *insertRecStmt;
    sqlite3_stmt *insertEnumStmt;
    sqlite3_stmt *insertTypedefStmt;
    sqlite3_stmt *insertGlobalVarStmt;
    sqlite3_stmt *insertMacroDefStmt;
};