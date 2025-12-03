# OpenAPI Specification Guide

This guide explains how to use the OpenAPI specification for the PicsApp API.

## Files

- **`openapi.yaml`** - OpenAPI 3.0 specification in YAML format

## Viewing the Specification

### Online Tools

1. **Swagger Editor** (Recommended)
   - Visit: https://editor.swagger.io/
   - Click "File" → "Import file" → Select `openapi.yaml`
   - View interactive API documentation

2. **Swagger UI**
   - Visit: https://swagger.io/tools/swagger-ui/
   - Upload the `openapi.yaml` file
   - Browse endpoints interactively

3. **Redoc**
   - Visit: https://redocly.com/reference-docs/redoc/
   - Upload the `openapi.yaml` file
   - View beautiful API documentation

### Command Line Tools

#### Using `swagger-codegen`
```bash
# Generate client SDKs
swagger-codegen generate -i docs/openapi.yaml -l javascript -o ./client-js
swagger-codegen generate -i docs/openapi.yaml -l go -o ./client-go
swagger-codegen generate -i docs/openapi.yaml -l python -o ./client-python
```

#### Using `openapi-generator`
```bash
# Generate client SDKs
openapi-generator generate -i docs/openapi.yaml -g javascript -o ./client-js
openapi-generator generate -i docs/openapi.yaml -g go -o ./client-go
openapi-generator generate -i docs/openapi.yaml -g python -o ./client-python
```

#### Using `redoc-cli`
```bash
# Generate static HTML documentation
npx redoc-cli bundle docs/openapi.yaml -o docs/api-docs.html
```

## Converting to JSON

If you need the specification in JSON format:

### Using Python
```bash
python3 -c "import yaml, json; print(json.dumps(yaml.safe_load(open('docs/openapi.yaml')), indent=2))" > docs/openapi.json
```

### Using Node.js
```bash
npm install -g yamljs
yaml2json docs/openapi.yaml > docs/openapi.json
```

### Using Online Converter
- Visit: https://www.json2yaml.com/convert-yaml-to-json
- Paste YAML content or upload file

## Validating the Specification

### Using `swagger-cli`
```bash
npm install -g swagger-cli
swagger-cli validate docs/openapi.yaml
```

### Using `spectral`
```bash
npm install -g @stoplight/spectral-cli
spectral lint docs/openapi.yaml
```

## Generating Code

### Server Stubs

Generate server stubs for various frameworks:

```bash
# Go server (Gin)
openapi-generator generate -i docs/openapi.yaml -g go-gin-server -o ./server-go

# Node.js server (Express)
openapi-generator generate -i docs/openapi.yaml -g nodejs-express-server -o ./server-node

# Python server (Flask)
openapi-generator generate -i docs/openapi.yaml -g python-flask -o ./server-python
```

### Client Libraries

Generate client libraries for various languages:

```bash
# JavaScript/TypeScript
openapi-generator generate -i docs/openapi.yaml -g typescript-axios -o ./client-ts

# Python
openapi-generator generate -i docs/openapi.yaml -g python -o ./client-python

# Go
openapi-generator generate -i docs/openapi.yaml -g go -o ./client-go
```

## API Testing

### Using `openapi-test`
```bash
npm install -g openapi-test
openapi-test docs/openapi.yaml --server http://localhost:8080
```

### Using Postman
1. Open Postman
2. Click "Import" → "File" → Select `openapi.yaml`
3. Postman will create a collection with all endpoints
4. Test endpoints directly from Postman

### Using Insomnia
1. Open Insomnia
2. Click "Create" → "Import/Export" → "Import Data" → "From File"
3. Select `openapi.yaml`
4. Insomnia will create requests for all endpoints

## Integration with CI/CD

### Validate in CI Pipeline

```yaml
# GitHub Actions example
- name: Validate OpenAPI Spec
  run: |
    npm install -g swagger-cli
    swagger-cli validate docs/openapi.yaml
```

### Generate Documentation in CI

```yaml
# GitHub Actions example
- name: Generate API Docs
  run: |
    npm install -g redoc-cli
    redoc-cli bundle docs/openapi.yaml -o docs/api-docs.html
```

## Specification Details

### OpenAPI Version
- **Version**: 3.0.3
- **Format**: YAML

### Covered Endpoints
- `POST /api/upload` - Upload picture
- `GET /api/pictures` - Get recent pictures
- `POST /api/pictures/{id}/like` - Like a picture
- `GET /api/presentation` - Get all pictures sorted by likes
- `GET /ws` - WebSocket connection (documented as HTTP endpoint)

### Schemas
- `Picture` - Picture object model
- `UploadResponse` - Upload response model
- `Error` - Error response model

## Notes

1. **WebSocket Support**: OpenAPI 3.0 doesn't fully support WebSocket specifications. The `/ws` endpoint is documented as an HTTP GET endpoint with a detailed description of the WebSocket protocol.

2. **Authentication**: Currently, the API has no authentication. The `securitySchemes` section is empty but ready for future implementation.

3. **Error Responses**: Most error responses return plain text, not JSON. This is documented in the specification.

4. **File Upload**: The upload endpoint uses `multipart/form-data` with a binary file field.

## Contributing

When updating the API:
1. Update `openapi.yaml` first
2. Update `API.md` to match
3. Validate the specification
4. Regenerate documentation if needed

