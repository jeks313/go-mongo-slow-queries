# Mongo Test Container

```
nerdctl run --name test-mongo -p 27017:27017 -e MONGO_INITDB_ROOT_USERNAME=root -e MONGO_INITDB_ROOT_PASSWORD=pass -d mongo:latest
```

```
nerdctl run -it --rm mongo mongo --host test-mongo -u root -p pass --authenticationDatabase admin
```