CREATE DATABASE slow_queries CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE slow_queries;
CREATE TABLE slow_queries
(
    id int unsigned primary key auto_increment not null,
    google_time datetime not null,
    sql_proxy  varchar(16) not null,
    thread_id bigint unsigned not null,
    server_id bigint unsigned not null,
    query_time float unsigned not null,
    lock_time float unsigned not null,
    rows_sent bigint unsigned not null,
    rows_examined bigint unsigned not null,
    google_timestamp bigint unsigned not null,
    sql_query        varchar(4096) not null
)