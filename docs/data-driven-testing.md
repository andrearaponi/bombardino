# Data-Driven Testing

Data-driven testing lets you run the same test with different data sets. Instead of creating 10 separate tests for 10 users, you create one test with 10 data rows.

## The Problem

You want to test creating users with different data. Without data-driven testing:

```json
{
  "tests": [
    {
      "name": "Create User 1",
      "method": "POST",
      "path": "/api/users",
      "body": {"name": "Mario", "age": 30}
    },
    {
      "name": "Create User 2",
      "method": "POST",
      "path": "/api/users",
      "body": {"name": "Luigi", "age": 28}
    },
    {
      "name": "Create User 3",
      "method": "POST",
      "path": "/api/users",
      "body": {"name": "Anna", "age": 25}
    }
  ]
}
```

This is repetitive and hard to maintain. Data-driven testing solves this.

## Inline Data

Add a `data` array to your test:

```json
{
  "name": "Create Users",
  "method": "POST",
  "path": "/api/users",
  "body": {
    "name": "${data.name}",
    "age": "${data.age}"
  },
  "data": [
    {"name": "Mario", "age": 30},
    {"name": "Luigi", "age": 28},
    {"name": "Anna", "age": 25}
  ]
}
```

Bombardino runs this test **3 times**, once for each data row:
1. First run: `name=Mario`, `age=30`
2. Second run: `name=Luigi`, `age=28`
3. Third run: `name=Anna`, `age=25`

## The `${data.field}` Syntax

Reference data fields using `${data.field_name}`:

| Reference | Gets |
|-----------|------|
| `${data.name}` | The `name` field from current data row |
| `${data.email}` | The `email` field from current data row |
| `${data.user.id}` | Nested field (if data has nested objects) |

## Type Preservation

Numbers and booleans keep their types:

```json
{
  "data": [
    {"name": "Mario", "age": 30, "active": true}
  ],
  "body": {
    "name": "${data.name}",
    "age": "${data.age}",
    "active": "${data.active}"
  }
}
```

The resulting JSON body is:
```json
{"name": "Mario", "age": 30, "active": true}
```

Not:
```json
{"name": "Mario", "age": "30", "active": "true"}
```

This is important for APIs that validate types.

## External Data Files

For larger data sets, use an external file.

### JSON File

Create `users.json`:
```json
[
  {"name": "Mario", "surname": "Rossi", "age": 30},
  {"name": "Luigi", "surname": "Verdi", "age": 28},
  {"name": "Anna", "surname": "Bianchi", "age": 25}
]
```

Reference it:
```json
{
  "name": "Create Users",
  "method": "POST",
  "path": "/api/users",
  "data_file": "users.json",
  "body": {
    "name": "${data.name}",
    "surname": "${data.surname}",
    "age": "${data.age}"
  }
}
```

### CSV File

Create `users.csv`:
```csv
name,surname,age
Mario,Rossi,30
Luigi,Verdi,28
Anna,Bianchi,25
```

Reference it:
```json
{
  "data_file": "users.csv"
}
```

CSV columns become field names.

## Complete Example: Testing Person API

Create multiple persons with inline data:

```json
{
  "name": "Data-Driven Person Creation",
  "global": {
    "base_url": "http://localhost:8080",
    "iterations": 1,
    "headers": {
      "Content-Type": "application/json"
    }
  },
  "tests": [
    {
      "name": "Create Multiple Persons",
      "method": "POST",
      "path": "/api/persons",
      "expected_status": [201],
      "data": [
        {
          "name": "Luigi",
          "surname": "Verdi",
          "age": 25,
          "height": 180.0,
          "weight": 80.0,
          "mail": "luigi.verdi@test.com"
        },
        {
          "name": "Anna",
          "surname": "Bianchi",
          "age": 28,
          "height": 165.5,
          "weight": 55.0,
          "mail": "anna.bianchi@test.com"
        },
        {
          "name": "Paolo",
          "surname": "Neri",
          "age": 35,
          "height": 178.0,
          "weight": 82.5,
          "mail": "paolo.neri@test.com"
        }
      ],
      "body": {
        "name": "${data.name}",
        "surname": "${data.surname}",
        "age": "${data.age}",
        "height": "${data.height}",
        "weight": "${data.weight}",
        "mail": "${data.mail}"
      },
      "assertions": [
        {
          "type": "status",
          "target": "response",
          "operator": "eq",
          "value": 201
        },
        {
          "type": "json_path",
          "target": "id",
          "operator": "exists",
          "value": ""
        }
      ],
      "extract": [
        {"name": "created_id", "source": "body", "path": "id"}
      ]
    },
    {
      "name": "Cleanup Created Persons",
      "method": "DELETE",
      "path": "/api/persons/${created_id}",
      "expected_status": [204],
      "depends_on": ["Create Multiple Persons"]
    }
  ]
}
```

This creates 3 persons, then deletes the last one.

## Combining with Iterations

Data rows multiply with iterations:

```json
{
  "global": {
    "iterations": 5
  },
  "tests": [
    {
      "name": "Create Users",
      "data": [
        {"name": "Mario"},
        {"name": "Luigi"}
      ]
    }
  ]
}
```

Total requests: 5 iterations × 2 data rows = **10 requests**

## Using Data in Different Places

Data can be used anywhere variables work:

**In path:**
```json
"path": "/api/users/${data.user_id}"
```

**In query string:**
```json
"path": "/api/search?name=${data.name}&age=${data.age}"
```

**In body:**
```json
"body": {
  "username": "${data.username}",
  "email": "${data.email}"
}
```

**In headers:**
```json
"headers": {
  "X-User-Token": "${data.token}"
}
```

## Assertions with Data-Driven Tests

Assertions run for each data row. Be careful with specific values:

**This works for all rows:**
```json
{
  "type": "status",
  "target": "response",
  "operator": "eq",
  "value": 201
}
```

**This might fail for some rows:**
```json
{
  "type": "json_path",
  "target": "name",
  "operator": "eq",
  "value": "Mario"
}
```

The second assertion only passes for the row with `name=Mario`.

## Tips

1. **Keep data organized**: Group similar data in arrays
2. **Use external files for large data**: Keep config files readable
3. **Test with small data first**: Verify the test works before scaling
4. **Check types in JSON data**: `"30"` is a string, `30` is a number
5. **Use meaningful field names**: `${data.user_email}` is clearer than `${data.e}`

## Debugging Data-Driven Tests

Use `-verbose` to see each data row:

```bash
bombardino -config test.json -verbose
```

Output shows:
```
[12:34:56] Running test "Create Users" with data row 1/3
[12:34:56] [a1b2c3d4] REQUEST POST /api/users
[12:34:56] Body: {"name":"Mario","age":30}
[12:34:57] [a1b2c3d4] RESPONSE 201 Created
...
[12:34:57] Running test "Create Users" with data row 2/3
...
```

## Next Steps

- [Configuration Reference](configuration-reference.md) - All data options
- [Assertions](assertions.md) - Validate data-driven responses
- [Tutorial: CRUD API](tutorial-crud-api.md) - Complete example
