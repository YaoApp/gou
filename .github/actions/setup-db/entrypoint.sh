#!/bin/sh

docker_run="docker run"

startMySQL() {
    VERSION=$1
    PORT="3306"
    if [ ! -z "$INPUT_PORT" ]; then
        PORT=$INPUT_PORT
    fi

    echo "Start MySQL $VERSION"
    docker_run="$docker_run -e MYSQL_ROOT_PASSWORD=$INPUT_PASSWORD -e MYSQL_USER=$INPUT_USER -e MYSQL_PASSWORD=$INPUT_PASSWORD"
    docker_run="$docker_run -e MYSQL_DATABASE=$INPUT_DB"
    docker_run="$docker_run -d -p $PORT:3306 mysql:$VERSION --port=3306"
    docker_run="$docker_run --character-set-server=utf8mb4 --collation-server=utf8mb4_general_ci"
    sh -c "$docker_run"

    DB_DSN="root:$INPUT_PASSWORD@tcp(127.0.0.1:$PORT)/$INPUT_DB?charset=utf8mb4&parseTime=True&loc=Local"
    echo "DSN=$DB_DSN" >> $GITHUB_ENV
    echo "DB_DRIVER=mysql" >> $GITHUB_ENV
    echo $DB_DSN
}

startPostgres() {
    VERSION=$1
    PORT="5432"
    if [ ! -z "$INPUT_PORT" ]; then
        PORT=$INPUT_PORT
    fi

    echo "Start Postgres $VERSION"
    docker_run="docker run"
    docker_run="$docker_run -e POSTGRES_DB=$INPUT_DB"
    docker_run="$docker_run -e POSTGRES_USER=$INPUT_USER"
    docker_run="$docker_run -e POSTGRES_PASSWORD=$INPUT_PASSWORD"
    docker_run="$docker_run -d -p $PORT:5432 postgres:$VERSION"

    DB_DSN="postgres://$INPUT_USER:$INPUT_PASSWORD@127.0.0.1:$PORT/$INPUT_DB?sslmode=disable"
    echo "DSN=$DB_DSN" >> $GITHUB_ENV
    echo "DB_DRIVER=postgres" >> $GITHUB_ENV
    echo $DB_DSN
}

startSQLite3() {
    echo "Start SQLite3"
    echo "DSN=file:$INPUT_DB.db?cache=shared" >> $GITHUB_ENV
    echo "DB_DRIVER=sqlite3" >> $GITHUB_ENV
}

# MySQL8.0, MySQL5.7, Postgres9.6, Postgres14, SQLite3
case $INPUT_KIND  in 
MySQL8.0)
    startMySQL 8.0
    ;;
MySQL5.7)
    startMySQL 5.7
    ;;
Postgres9.6)
    startPostgres 9.6
    ;;
Postgres14)
    startPostgres 14
    ;;
SQLite3)
    startSQLite3 
    ;;
esac
