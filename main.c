#include "sqlite3.h"
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

#define panic(msg)                                                             \
  fprintf(stderr, "runtime error: %s\n", msg);                                 \
  exit(1);

typedef enum {
  TEST = 0,
  REFEREES,
  CLUBS,
} table_name;

static const char *const tables[] = {
    [TEST] = "test",
    [REFEREES] = "referees",
    [CLUBS] = "clubs",
};

sqlite3 *connect();
void execute_sql(sqlite3 *db, char *sql);
void show_table(sqlite3 *db, table_name name);
static int callback(void *data, int argc, char **argv, char **azColName);

int main(void) {
  sqlite3 *db = connect();
  show_table(db, TEST);
  sqlite3_close(db);
  exit(0);
}

sqlite3 *connect() {
  sqlite3 *db;
  if (sqlite3_open("database.db", &db) != SQLITE_OK) {
    panic(sqlite3_errmsg(db));
  }
  return db;
}

void execute_sql(sqlite3 *db, char *sql) {
  char *errmsg;
  printf("executing: %s\n", sql);
  if (sqlite3_exec(db, sql, callback, 0, &errmsg) != SQLITE_OK) {
    panic(errmsg);
  };
};

static int callback(void *data, int argc, char **argv, char **azColName) {
  int i;
  for (i = 0; i < argc; i++) {
    printf("%s = %s\n", azColName[i], argv[i] ? argv[i] : "NULL");
  }
  printf("\n");
  return 0;
}
void show_table(sqlite3 *db, table_name name) {
  char sql[64];
  snprintf(sql, 64, "select * from %s", tables[name]);
  execute_sql(db, sql);
};
