# Go Server

It is like json-server but written in go

## How it works

The idea is that you would have a db.json file somewhere  and would give the
location as an argument to the server. The file would look something like this:

```json
{
  "person": [
    {
      "id": 1,
      "name": "Fitz",
      "age": 22
    }
  ]
}
```

By doing:

```console
go-server --watch ./db.json
```

Would be created endpoints for person, like:

```
GET localhost:3000/person
GET localhost:3000/person/1
POST localhost:3000/person
```
