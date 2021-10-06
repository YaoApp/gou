#!/bin/sh

docker_run="docker run"

startMySQL() {
    VERSION=$1
    echo "Start MySQL $VERSION"
}

startPostgres() {
    VERSION=$1
    echo "Start Postgres $VERSION"
}

startSQLite3() {
    echo "Start SQLite3"
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
    startMySQL 9.6
    ;;
Postgres14)
    startMySQL 14
    ;;
SQLite3)
    startSQLite3 
    ;;
esac

# if [ -n "$INPUT_PASSWORD" ]; then
#   echo "Root password not empty, use root superuser"

#   docker_run="$docker_run -e MYSQL_ROOT_PASSWORD=$INPUT_PASSWORD"
# elif [ -n "$INPUT_USER" ]; then
#   if [ -z "$INPUT_PASSWORD" ]; then
#     echo "The password must not be empty when user exists"
#     exit 1
#   fi

#   echo "Use specified user and password"

#   docker_run="$docker_run -e MYSQL_RANDOM_ROOT_PASSWORD=true -e MYSQL_USER=$INPUT_USER -e MYSQL_PASSWORD=$INPUT_PASSWORD"
# else
#   echo "Both root password and superuser are empty, must contains one superuser"
#   exit 1
# fi

# if [ -n "$INPUT_DB" ]; then
#   echo "Use specified database"

#   docker_run="$docker_run -e MYSQL_DATABASE=$INPUT_DB"
# fi

# docker_run="$docker_run -d -p $INPUT_HOST_PORT:$INPUT_CONTAINER_PORT mysql:$INPUT_VERSION --port=$INPUT_PORT"
# docker_run="$docker_run --character-set-server=$INPUT_CHARACTER_SET_SERVER --collation-server=$INPUT_COLLATION_SERVER"