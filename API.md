# API

## Insert new fortunes

```
POST /
```

Accepts a plain text request body containing fortunes, separated by `%` as per the original format of the Unix `fortune` command. The fortunes are stored in bulk.

- **Request**

  - **Headers**:
    - `Content-Type: text/plain`
  - **Body Example**:
    ```text
    Fortune favors the bold.
    %
    You will have a pleasant surprise.
    ```

- **Responses**
  - ✅ **`201 Created`** – Fortunes successfully inserted.
    - **Header**: `X-Inserted-Count` (Number of inserted fortunes)
  - ⚠️ **`400 Bad Request`** – No valid fortunes provided.
  - 🚫 **`413 Payload Too Large`** – Exceeds 1MB limit.
  - ❌ **`415 Unsupported Media Type`** – Must be `text/plain`.

---

## Get a random fortune

```
GET /
```

Returns a randomly selected fortune from the database.

- **Responses**
  - ✅ **`200 OK`** – Fortune retrieved successfully.
    - **Example**:
      ```text
      You will have a pleasant surprise.
      ```
  - ❌ **`404 Not Found`** – No fortunes in the database.

---

For the full OpenAPI 3.0.3 specification, see [`etc/openapi.yaml`](./etc/openapi.yaml).
