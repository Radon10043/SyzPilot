#include <iostream>
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
    std::string name; /* enum name       */
    std::string file; /* file path      */
    int line;         /* line number    */
    std::string code; /* enum code      */
};

struct TypedefInfo {
    std::string typ;   /* old name       */
    std::string def;   /* new name       */
    std::string file;  /* file path      */
    int line;          /* line number    */
    std::string code;  /* typedef code   */
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

        createFuncTable();
        createRecTable();
        createEnumTable();
        createTypedefTable();
    }

    ~DatabaseManager() {
        sqlite3_finalize(insertFuncStmt);
        sqlite3_finalize(insertRecStmt);
        sqlite3_finalize(insertEnumStmt);
        sqlite3_finalize(insertTypedefStmt);
        sqlite3_close(db);
    }

    /**
     * create functions table
     */
    void createFuncTable() {
        const char *funcSql = "CREATE TABLE IF NOT EXISTS functions ("
                              "id INTEGER PRIMARY KEY AUTOINCREMENT, "
                              "name TEXT NOT NULL, "
                              "file TEXT NOT NULL, "
                              "line INTEGER NOT NULL, "
                              "code TEXT NOT NULL);";
        if (sqlite3_exec(db, funcSql, 0, 0, 0) != SQLITE_OK) {
            llvm::errs() << "Failed to create table: " << sqlite3_errmsg(db) << "\n";
            exit(1);
        }
        const char *insertFuncSql = "INSERT INTO functions (name, file, line, code) VALUES (?, ?, ?, ?);";
        if (sqlite3_prepare_v2(db, insertFuncSql, -1, &insertFuncStmt, nullptr) != SQLITE_OK) {
            llvm::errs() << "Failed to prepare statement: " << sqlite3_errmsg(db) << "\n";
            exit(1);
        }
    }

    /**
     * create records table
     */
    void createRecTable() {
        const char *recSql = "CREATE TABLE IF NOT EXISTS records ("
                             "id INTEGER PRIMARY KEY AUTOINCREMENT, "
                             "name TEXT NOT NULL, "
                             "type TEXT NOT NULL, "
                             "file TEXT NOT NULL, "
                             "line INTEGER NOT NULL, "
                             "code TEXT NOT NULL);";
        if (sqlite3_exec(db, recSql, 0, 0, 0) != SQLITE_OK) {
            llvm::errs() << "Failed to create table: " << sqlite3_errmsg(db) << "\n";
            exit(1);
        }
        const char *insertRecSql = "INSERT INTO records (name, type, file, line, code) VALUES (?, ?, ?, ?, ?);";
        if (sqlite3_prepare_v2(db, insertRecSql, -1, &insertRecStmt, nullptr) != SQLITE_OK) {
            llvm::errs() << "Failed to prepare statement: " << sqlite3_errmsg(db) << "\n";
            exit(1);
        }
    }

    /**
     * create enums table
     */
    void createEnumTable() {
        const char *enumSql = "CREATE TABLE IF NOT EXISTS enums ("
                              "id INTEGER PRIMARY KEY AUTOINCREMENT, "
                              "name TEXT NOT NULL, "
                              "file TEXT NOT NULL, "
                              "line INTEGER NOT NULL, "
                              "code TEXT NOT NULL);";
        if (sqlite3_exec(db, enumSql, 0, 0, 0) != SQLITE_OK) {
            llvm::errs() << "Failed to create table: " << sqlite3_errmsg(db) << "\n";
            exit(1);
        }
        const char *insertEnumSql = "INSERT INTO enums (name, file, line, code) VALUES (?, ?, ?, ?);";
        if (sqlite3_prepare_v2(db, insertEnumSql, -1, &insertEnumStmt, nullptr) != SQLITE_OK) {
            llvm::errs() << "Failed to prepare statement: " << sqlite3_errmsg(db) << "\n";
            exit(1);
        }
    }

    /**
     * create typedefs table
     */
    void createTypedefTable() {
        const char *typedefSql = "CREATE TABLE IF NOT EXISTS typedefs ("
                                 "id INTEGER PRIMARY KEY AUTOINCREMENT, "
                                 "type TEXT NOT NULL, "
                                 "define TEXT NOT NULL, "
                                 "file TEXT NOT NULL, "
                                 "line INTEGER NOT NULL, "
                                 "code TEXT NOT NULL);";
        if (sqlite3_exec(db, typedefSql, 0, 0, 0) != SQLITE_OK) {
            llvm::errs() << "Failed to create table: " << sqlite3_errmsg(db) << "\n";
            exit(1);
        }
        const char *insertTypedefSql = "INSERT INTO typedefs (type, define, file, line, code) VALUES (?, ?, ?, ?, ?);";
        if (sqlite3_prepare_v2(db, insertTypedefSql, -1, &insertTypedefStmt, nullptr) != SQLITE_OK) {
            llvm::errs() << "Failed to prepare statement: " << sqlite3_errmsg(db) << "\n";
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
                llvm::errs() << "Insert error: " << sqlite3_errmsg(db) << "\n";
            sqlite3_reset(insertFuncStmt);
        }
        if (sqlite3_exec(db, "COMMIT;", nullptr, nullptr, &err) != SQLITE_OK) {
            llvm::errs() << "Commit error: " << err << "\n";
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
                llvm::errs() << "Insert error: " << sqlite3_errmsg(db) << "\n";
            sqlite3_reset(insertRecStmt);
        }
        if (sqlite3_exec(db, "COMMIT;", nullptr, nullptr, &err) != SQLITE_OK) {
            llvm::errs() << "Commit error: " << err << "\n";
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
            sqlite3_bind_text(insertEnumStmt, 1, enm.name.c_str(), -1, SQLITE_STATIC);
            sqlite3_bind_text(insertEnumStmt, 2, enm.file.c_str(), -1, SQLITE_STATIC);
            sqlite3_bind_int(insertEnumStmt, 3, enm.line);
            sqlite3_bind_text(insertEnumStmt, 4, enm.code.c_str(), -1, SQLITE_STATIC);
            if (sqlite3_step(insertEnumStmt) != SQLITE_DONE)
                llvm::errs() << "Insert error: " << sqlite3_errmsg(db) << "\n";
            sqlite3_reset(insertEnumStmt);
        }
        if (sqlite3_exec(db, "COMMIT;", nullptr, nullptr, &err) != SQLITE_OK) {
            llvm::errs() << "Commit error: " << err << "\n";
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
                llvm::errs() << "Insert error: " << sqlite3_errmsg(db) << "\n";
            sqlite3_reset(insertTypedefStmt);
        }
        if (sqlite3_exec(db, "COMMIT;", nullptr, nullptr, &err) != SQLITE_OK) {
            llvm::errs() << "Commit error: " << err << "\n";
            sqlite3_free(err);
        }
    }

private:
    sqlite3 *db;
    sqlite3_stmt *insertFuncStmt;
    sqlite3_stmt *insertRecStmt;
    sqlite3_stmt *insertEnumStmt;
    sqlite3_stmt *insertTypedefStmt;
};