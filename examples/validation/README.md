# Validation example

This example demonstrates the request and response validation capabilities of Meridian.

## Files

- [openapi.yaml](openapi.yaml) - OpenAPI specification
- [valid-request.json](valid-request.json) - A valid request example
- [invalid-request.json](invalid-request.json) - An invalid request example
- [valid-response.json](valid-response.json) - A valid response example

## Validating requests

### Valid request

```bash
meridian validate --spec examples/validation/openapi.yaml \
  --request examples/validation/valid-request.json
```

Expected output:

```
Request is valid
```

### Invalid request

```bash
meridian validate --spec examples/validation/openapi.yaml \
  --request examples/validation/invalid-request.json
```

Expected output:

```
Request validation failed:
  - name: string too short, minimum 2
  - email: invalid email format
```

## Validating responses

```bash
meridian validate --spec examples/validation/openapi.yaml \
  --response examples/validation/valid-response.json
```

Expected output:

```
Response is valid
```
