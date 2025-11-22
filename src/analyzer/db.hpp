#include <iostream>
#include <sqlite3.h>
#include <string>

#include <llvm/Support/raw_ostream.h>

class DatabaseManager {
public:
    DatabaseManager() = default;

    DatabaseManager(const std::string &dbPath) : db(nullptr), stmt(nullptr) {
        if (sqlite3_open(dbPath.c_str(), &db) != SQLITE_OK) {
            llvm::errs() << "Cannot open database: " << sqlite3_errmsg(db) << "\n";
            exit(1);
        }

        // Enable WAL to speed up writes
        char *err = nullptr;
        sqlite3_exec(db, "PRAGMA synchronous = OFF; PRAGMA journal_mode = WAL;", nullptr, nullptr, &err);

        const char *sql = "CREATE TABLE IF NOT EXISTS functions ("
                          "id INTEGER PRIMARY KEY AUTOINCREMENT, "
                          "name TEXT NOT NULL, "
                          "file TEXT NOT NULL, "
                          "line INTEGER NOT NULL, "
                          "code TEXT NOT NULL);";
        if (sqlite3_exec(db, sql, 0, 0, 0) != SQLITE_OK) {
            llvm::errs() << "Failed to create table: " << sqlite3_errmsg(db) << "\n";
            sqlite3_free(err);
            exit(1);
        }

        const char *insertSQL = "INSERT INTO functions (name, file, line, code) VALUES (?, ?, ?, ?);";
        if (sqlite3_prepare_v2(db, insertSQL, -1, &stmt, nullptr) != SQLITE_OK) {
            llvm::errs() << "Failed to prepare statement: " << sqlite3_errmsg(db) << "\n";
            exit(1);
        }
    }

    ~DatabaseManager() {
        sqlite3_finalize(stmt);
        sqlite3_close(db);
    }

    /**
     * begin a transaction
     */
    void beginTransaction() {
        sqlite3_exec(db, "BEGIN TRANSACTION;", nullptr, nullptr, nullptr);
    }

    /**
     * commit a transaction
     */
    void commitTransaction() {
        sqlite3_exec(db, "COMMIT;", nullptr, nullptr, nullptr);
    }

    /**
     * insert a function record into the database
     */
    void insertFunction(const std::string &name, const std::string &file, int line, const std::string &code) {
        sqlite3_bind_text(stmt, 1, name.c_str(), -1, SQLITE_STATIC);
        sqlite3_bind_text(stmt, 2, file.c_str(), -1, SQLITE_STATIC);
        sqlite3_bind_int(stmt, 3, line);
        sqlite3_bind_text(stmt, 4, code.c_str(), -1, SQLITE_STATIC);

        if (sqlite3_step(stmt) != SQLITE_DONE) {
            llvm::errs() << "Failed to execute statement: " << sqlite3_errmsg(db) << "\n";
        }

        sqlite3_reset(stmt);
    }

private:
    sqlite3 *db;
    sqlite3_stmt *stmt;
};