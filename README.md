# Mongo Slow Query Exporter

I wanted a way of showing the currently running slow queries in real time. The issue with slow query log parsing
is that those logs are only written after the queries have completed.

This exporter runs `db.currentOp()` on an interval and emits metrics about running queries it sees. After a query
has completed it updates a histogram with the running query time. This is far more efficient than writing a log
parser for the slow query log.

This exporter will:

- Show you in realtime the running slow queries.
- Captures what users and databases and collections the slow queries are running against.
- Keeps a history of the last 1000 slow queries run for quick examination.
- Provides an endpoint to see the running queries without having to login to the Mongo server.

## Mongo Test Container

```
docker run --name test-mongo -p 27017:27017 -e MONGO_INITDB_ROOT_USERNAME=root -e MONGO_INITDB_ROOT_PASSWORD=pass -d mongo:latest
```

```
docker run -it --rm mongo mongo --host test-mongo -u root -p pass --authenticationDatabase admin
```
