# Product API

A Go + Gin implementation of the Product endpoints in `api.yaml`. Products are stored in memory.

| Method | Path | Responses |
|---|---|---|
| `GET` | `/products/{productId}` | `200`, `404`, `500` |
| `POST` | `/products/{productId}/details` | `204`, `400`, `404`, `500` |

## Running

With Go:
```bash
go mod tidy
go run product-service.go
```

With Docker:
```bash
docker build -t product-api .
docker run --rm -p 8080:8080 product-api
```

The server listens on `http://localhost:8080`.

## Behaviour

- `POST` creates the product, or replaces it if it already exists.
- All six Product fields are required and must follow the limits in `api.yaml`. The `product_id` in the body must match the one in the path. Otherwise the response is `400`.
- A `productId` in the path that isn't a positive 32-bit integer (e.g. `abc`, `0`) returns `404`, since it can't identify a product.
- Any request with the header `X-Simulate-Error: true` returns `500`, so the error response can be tested.

## Examples

One example for every response code of each endpoint. Run them in order: the `GET 200` example needs the product created by the `POST 204` example.

| Endpoint | Codes shown |
|---|---|
| `POST /products/{productId}/details` | 204, 400, 404, 500 |
| `GET /products/{productId}` | 200, 404, 500 |

### POST /products/{productId}/details

**204 No Content: product created**
```bash
curl -i -X POST http://localhost:8080/products/12345/details \
  -H "Content-Type: application/json" \
  -d '{"product_id":12345,"sku":"ABC-123-XYZ","manufacturer":"Acme Corporation","category_id":456,"weight":1250,"some_other_id":789}'
```
```
HTTP/1.1 204 No Content
```

**400 Bad Request: missing required fields**
```bash
curl -i -X POST http://localhost:8080/products/12345/details \
  -H "Content-Type: application/json" \
  -d '{"product_id":12345}'
```
```
HTTP/1.1 400 Bad Request
Content-Type: application/json; charset=utf-8

{"error":"INVALID_INPUT","message":"The provided input data is invalid","details":"Key: 'Product.SKU' Error:Field validation for 'SKU' failed on the 'required' tag\nKey: 'Product.Manufacturer' Error:Field validation for 'Manufacturer' failed on the 'required' tag\nKey: 'Product.CategoryID' Error:Field validation for 'CategoryID' failed on the 'required' tag\nKey: 'Product.Weight' Error:Field validation for 'Weight' failed on the 'required' tag\nKey: 'Product.SomeOtherID' Error:Field validation for 'SomeOtherID' failed on the 'required' tag"}
```

**404 Not Found: invalid product ID**
```bash
curl -i -X POST http://localhost:8080/products/abc/details \
  -H "Content-Type: application/json" \
  -d '{"product_id":12345,"sku":"ABC-123-XYZ","manufacturer":"Acme Corporation","category_id":456,"weight":1250,"some_other_id":789}'
```
```
HTTP/1.1 404 Not Found
Content-Type: application/json; charset=utf-8

{"error":"NOT_FOUND","message":"Product not found","details":"product ID must be a positive 32-bit integer"}
```

**500 Internal Server Error: simulated failure**
```bash
curl -i -X POST http://localhost:8080/products/12345/details \
  -H "Content-Type: application/json" \
  -H "X-Simulate-Error: true" \
  -d '{"product_id":12345,"sku":"ABC-123-XYZ","manufacturer":"Acme Corporation","category_id":456,"weight":1250,"some_other_id":789}'
```
```
HTTP/1.1 500 Internal Server Error
Content-Type: application/json; charset=utf-8

{"error":"INTERNAL_ERROR","message":"Internal server error"}
```

### GET /products/{productId}

**200 OK: product found**
```bash
curl -i http://localhost:8080/products/12345
```
```
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8

{"product_id":12345,"sku":"ABC-123-XYZ","manufacturer":"Acme Corporation","category_id":456,"weight":1250,"some_other_id":789}
```

**404 Not Found: product doesn't exist**
```bash
curl -i http://localhost:8080/products/99999
```
```
HTTP/1.1 404 Not Found
Content-Type: application/json; charset=utf-8

{"error":"NOT_FOUND","message":"Product not found"}
```

**500 Internal Server Error: simulated failure**
```bash
curl -i http://localhost:8080/products/12345 -H "X-Simulate-Error: true"
```
```
HTTP/1.1 500 Internal Server Error
Content-Type: application/json; charset=utf-8

{"error":"INTERNAL_ERROR","message":"Internal server error"}
```

## Postman

The exported Postman collection `product-api.postman_collection.json` contains every request above, plus extra 400 cases (malformed JSON, invalid values, path/body ID mismatch). Import it into Postman, then right-click the collection and choose **Run collection**. Keep the default order.
